package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

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
	uploadErrors := make([]models.UploadErrorInfo, 0)
	for _, file := range files {
		// Open file
		src, err := file.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("failed to open file: %v", err),
				Stage:    "file_open",
			})
			continue
		}

		// Read file data
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("failed to read file data: %v", err),
				Stage:    "file_read",
			})
			continue
		}

		// Detect MIME type from file extension
		mimeType := "image/jpeg" // Default
		if len(file.Filename) > 0 {
			ext := file.Filename[len(file.Filename)-4:]
			if ext == ".png" || ext == "PNG" {
				mimeType = "image/png"
			} else if ext == ".heic" || ext == "HEIC" {
				mimeType = "image/heic"
			} else if ext == ".cr2" || ext == "CR2" {
				mimeType = "image/x-canon-cr2"
			}
		}

		// Create bracket in AutoEnhance
		var bracket *autoenhance.BracketCreatedOut
		err = h.autoenhanceClient.RetryWithBackoff(func() error {
			var err error
			bracket, err = h.autoenhanceClient.CreateBracket(autoenhance.BracketIn{
				Name:    file.Filename,
				OrderID: order.ID.String(),
			})
			return err
		}, 3)
		if err != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("failed to create bracket in AutoEnhance: %v", err),
				Stage:    "create_bracket",
			})
			continue
		}

		// Check if upload URL is provided
		if bracket.UploadURL == "" {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    "AutoEnhance did not provide an upload URL in the bracket creation response",
				Stage:    "create_bracket",
			})
			continue
		}

		// Upload to bracket upload URL
		err = h.autoenhanceClient.RetryWithBackoff(func() error {
			return h.autoenhanceClient.UploadFile(bracket.UploadURL, data, mimeType)
		}, 3)
		if err != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("failed to upload file to AutoEnhance storage: %v", err),
				Stage:    "upload",
			})
			continue
		}

		// Verify the upload by checking the bracket status with AutoEnhance
		// AutoEnhance processes uploads asynchronously, so we wait a bit and retry
		var verifiedBracket *autoenhance.BracketOut
		verified := false
		maxRetries := 3
		retryDelay := 500 * time.Millisecond
		
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(retryDelay)
			}
			
			var err error
			verifiedBracket, err = h.autoenhanceClient.GetBracket(bracket.BracketID)
			if err != nil {
				if attempt == maxRetries-1 {
					// Last attempt failed - log warning but don't fail upload
					uploadErrors = append(uploadErrors, models.UploadErrorInfo{
						Filename: file.Filename,
						Error:    fmt.Sprintf("upload HTTP succeeded but verification failed after %d attempts: %v", maxRetries, err),
						Stage:    "verify",
					})
				}
				continue
			}
			
			// Check if bracket is marked as uploaded
			if verifiedBracket.IsUploaded {
				verified = true
				// Update our DB with the actual status from AutoEnhance
				if verifiedBracket.ImageID != "" && verifiedBracket.ImageID != bracket.ImageID {
					bracket.ImageID = verifiedBracket.ImageID
				}
				break
			}
		}
		
		// If still not verified after retries, log a warning
		if !verified && verifiedBracket != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("upload HTTP succeeded (200/204) but AutoEnhance reports is_uploaded=false after %d verification attempts. This may be normal - AutoEnhance processes uploads asynchronously. BracketID: %s", maxRetries, bracket.BracketID),
				Stage:    "verify",
			})
		}

		// Store bracket in database
		// Mark as uploaded since the HTTP request succeeded (200/204)
		// AutoEnhance will update the status asynchronously
		bracketModel := &models.Bracket{
			ID:         uuid.New(),
			OrderID:    orderID,
			BracketID:  bracket.BracketID,
			Filename:   file.Filename,
			IsUploaded: true, // HTTP upload succeeded, so mark as uploaded
			Metadata:   json.RawMessage("{}"), // Initialize with empty JSON object
		}
		if bracket.UploadURL != "" {
			bracketModel.UploadURL = sql.NullString{String: bracket.UploadURL, Valid: true}
		}
		if bracket.ImageID != "" {
			bracketModel.ImageID = sql.NullString{String: bracket.ImageID, Valid: true}
		}
		// If bracket has metadata from AutoEnhance, use it
		if bracket.Metadata != nil && len(bracket.Metadata) > 0 {
			if metadataBytes, err := json.Marshal(bracket.Metadata); err == nil {
				bracketModel.Metadata = json.RawMessage(metadataBytes)
			}
		}
		if err := h.dbClient.CreateBracket(bracketModel); err != nil {
			uploadErrors = append(uploadErrors, models.UploadErrorInfo{
				Filename: file.Filename,
				Error:    fmt.Sprintf("upload succeeded but failed to save bracket to database: %v", err),
				Stage:    "database",
			})
			// Continue anyway since the upload succeeded
		}

		uploadedFiles = append(uploadedFiles, models.FileInfo{
			Filename: file.Filename,
			Size:     file.Size,
		})
	}

	if len(uploadedFiles) == 0 {
		errorMsg := "failed to upload any files"
		if len(uploadErrors) > 0 {
			errorDetails := make([]string, len(uploadErrors))
			for i, e := range uploadErrors {
				errorDetails[i] = fmt.Sprintf("%s [%s]: %s", e.Filename, e.Stage, e.Error)
			}
			errorMsg += ": " + fmt.Sprintf("%v", errorDetails)
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

	// Include errors in response if any files failed
	response := models.UploadResponse{
		OrderID: orderID.String(),
		Files:   uploadedFiles,
		Status:  "uploaded",
	}
	if len(uploadErrors) > 0 {
		response.Errors = uploadErrors
		// Also log to database with detailed error info
		errorDetails := make([]string, len(uploadErrors))
		for i, e := range uploadErrors {
			errorDetails[i] = fmt.Sprintf("%s [%s]: %s", e.Filename, e.Stage, e.Error)
		}
		errorMsg := fmt.Sprintf("Some files had issues: %v", errorDetails)
		h.dbClient.UpdateOrderError(orderID, errorMsg)
	}

	c.JSON(http.StatusOK, response)
}
