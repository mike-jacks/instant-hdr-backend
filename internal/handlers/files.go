package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type FilesHandler struct {
	dbClient *supabase.DatabaseClient
}

func NewFilesHandler(dbClient *supabase.DatabaseClient) *FilesHandler {
	return &FilesHandler{
		dbClient: dbClient,
	}
}

// GetFiles godoc
// @Summary     Get order files
// @Description Returns a list of all processed files associated with an order, including their Supabase Storage URLs
// @Tags        files
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} models.FilesResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/files [get]
func (h *FilesHandler) GetFiles(c *gin.Context) {
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

	// Get processed files (final images) only
	files, err := h.dbClient.GetOrderFiles(orderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get files",
			Message: err.Error(),
		})
		return
	}

	fileResponses := make([]models.FileResponse, len(files))
	for i, file := range files {
		fileSize := int64(0)
		if file.FileSize.Valid {
			fileSize = file.FileSize.Int64
		}
		fileResponses[i] = models.FileResponse{
			ID:         file.ID.String(),
			Filename:   file.Filename,
			StorageURL: file.StorageURL,
			FileSize:   fileSize,
			MimeType:   file.MimeType,
			IsFinal:    file.IsFinal,
			CreatedAt:  file.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, models.FilesResponse{Files: fileResponses})
}

// GetBrackets godoc
// @Summary     Get uploaded brackets
// @Description Returns a list of all uploaded brackets (raw images) for an order
// @Tags        brackets
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} models.BracketsResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/brackets [get]
func (h *FilesHandler) GetBrackets(c *gin.Context) {
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
	_, err = h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Get uploaded brackets
	brackets, err := h.dbClient.GetBracketsByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get brackets",
			Message: err.Error(),
		})
		return
	}

	bracketResponses := make([]models.BracketResponse, len(brackets))
	for i, bracket := range brackets {
		bracketResponses[i] = models.BracketResponse{
			ID:         bracket.ID.String(),
			BracketID:  bracket.BracketID,
			Filename:   bracket.Filename,
			IsUploaded: bracket.IsUploaded,
			CreatedAt:  bracket.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, models.BracketsResponse{Brackets: bracketResponses})
}
