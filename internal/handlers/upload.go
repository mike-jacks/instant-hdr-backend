package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type UploadHandler struct {
	autoenhanceClient *autoenhance.Client
	dbClient          *supabase.DatabaseClient
	realtimeClient    *supabase.RealtimeClient
}

func NewUploadHandler(autoenhanceClient *autoenhance.Client, dbClient *supabase.DatabaseClient, realtimeClient *supabase.RealtimeClient) *UploadHandler {
	return &UploadHandler{
		autoenhanceClient: autoenhanceClient,
		dbClient:          dbClient,
		realtimeClient:    realtimeClient,
	}
}

// Upload godoc
// @Summary     Upload images to order
// @Description Uploads multiple bracketed images to an AutoEnhance AI order. All images in a single upload are expected to be bracketed images of the same shot.
// @Tags        upload
// @Accept      multipart/form-data
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Param       images formData file true "Bracketed images (multiple files allowed)"
// @Success     200 {object} models.UploadResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/upload [post]
func (h *UploadHandler) Upload(c *gin.Context) {
	if h.dbClient == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database not available"})
		return
	}

	userIDStr, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid user id"})
		return
	}

	orderIDStr := c.Param("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid order id"})
		return
	}

	// Verify order belongs to user
	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Set max memory for multipart form (32MB)
	err = c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse multipart form",
			Message: err.Error(),
		})
		return
	}

	// Parse multipart form
	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse multipart form",
			Message: "multipart form is nil",
		})
		return
	}

	// Try multiple common field names
	var files []*multipart.FileHeader
	fieldNames := []string{"images", "image", "files", "file", "photos", "photo"}
	for _, fieldName := range fieldNames {
		if f := form.File[fieldName]; len(f) > 0 {
			files = f
			break
		}
	}

	if len(files) == 0 {
		// Get all available field names for debugging
		availableFields := make([]string, 0, len(form.File))
		for fieldName := range form.File {
			availableFields = append(availableFields, fieldName)
		}
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "no files uploaded",
			Message: fmt.Sprintf("please provide files with one of these field names: %v. Available fields in request: %v", fieldNames, availableFields),
		})
		return
	}

	// Publish upload_started event
	h.realtimeClient.PublishOrderEvent(orderID, "upload_started",
		supabase.UploadStartedPayload(orderID, len(files)))

	// Update status
	h.dbClient.UpdateOrderStatus(orderID, "uploading", 0)

	// Create brackets and upload files
	uploadedFiles := make([]models.FileInfo, 0)
	uploadErrors := make([]string, 0)
	for _, file := range files {
		// Open file
		src, err := file.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to open: %v", file.Filename, err))
			continue
		}

		// Read file data
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to read: %v", file.Filename, err))
			continue
		}

		// Create bracket in AutoEnhance
		var bracket *autoenhance.BracketCreatedOut
		err = h.autoenhanceClient.RetryWithBackoff(func() error {
			var err error
			bracket, err = h.autoenhanceClient.CreateBracket(autoenhance.BracketIn{
				Name:    file.Filename,
				OrderID: order.AutoEnhanceOrderID,
			})
			return err
		}, 3)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to create bracket: %v", file.Filename, err))
			continue
		}

		// Upload to bracket upload URL
		err = h.autoenhanceClient.RetryWithBackoff(func() error {
			return h.autoenhanceClient.UploadFile(bracket.UploadURL, data)
		}, 3)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to upload: %v", file.Filename, err))
			continue
		}

		// Store bracket in database
		bracketModel := &models.Bracket{
			ID:         uuid.New(),
			OrderID:    orderID,
			BracketID:  bracket.BracketID,
			Filename:   file.Filename,
			IsUploaded: true,
		}
		if bracket.UploadURL != "" {
			bracketModel.UploadURL = sql.NullString{String: bracket.UploadURL, Valid: true}
		}
		if err := h.dbClient.CreateBracket(bracketModel); err != nil {
			// Log error but continue
		}

		uploadedFiles = append(uploadedFiles, models.FileInfo{
			Filename: file.Filename,
			Size:     file.Size,
		})
	}

	if len(uploadedFiles) == 0 {
		errorMsg := "failed to upload any files"
		if len(uploadErrors) > 0 {
			errorMsg += ": " + fmt.Sprintf("%v", uploadErrors)
		}
		h.dbClient.UpdateOrderError(orderID, errorMsg)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to upload files",
			Message: errorMsg,
		})
		return
	}

	// Update status
	h.dbClient.UpdateOrderStatus(orderID, "uploaded", 0)

	// Publish upload_completed event
	h.realtimeClient.PublishOrderEvent(orderID, "upload_completed",
		supabase.UploadCompletedPayload(orderID, len(uploadedFiles)))

	c.JSON(http.StatusOK, models.UploadResponse{
		OrderID: orderID.String(),
		Files:   uploadedFiles,
		Status:  "uploaded",
	})
}
