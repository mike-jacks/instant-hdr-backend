package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type FilesHandler struct {
	dbClient          *supabase.DatabaseClient
	autoenhanceClient *autoenhance.Client
}

func NewFilesHandler(dbClient *supabase.DatabaseClient, autoenhanceClient *autoenhance.Client) *FilesHandler {
	return &FilesHandler{
		dbClient:          dbClient,
		autoenhanceClient: autoenhanceClient,
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

	// Get brackets from our database
	dbBrackets, err := h.dbClient.GetBracketsByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get brackets",
			Message: err.Error(),
		})
		return
	}

	// Also fetch brackets from AutoEnhance to verify and sync
	var autoenhanceBrackets map[string]*autoenhance.BracketOut
	if h.autoenhanceClient != nil {
		orderBrackets, err := h.autoenhanceClient.GetOrderBrackets(orderID.String())
		if err == nil && orderBrackets != nil {
			// Create a map for quick lookup
			autoenhanceBrackets = make(map[string]*autoenhance.BracketOut)
			for i := range orderBrackets.Brackets {
				b := &orderBrackets.Brackets[i]
				autoenhanceBrackets[b.BracketID] = b
			}
		}
	}

	// Combine data from both sources, prioritizing AutoEnhance status and metadata
	bracketResponses := make([]models.BracketResponse, len(dbBrackets))
	for i, bracket := range dbBrackets {
		response := models.BracketResponse{
			ID:         bracket.ID.String(),
			BracketID:  bracket.BracketID,
			Filename:   bracket.Filename,
			IsUploaded: bracket.IsUploaded,
			CreatedAt:  bracket.CreatedAt,
		}

		// Start with our database metadata (includes group_id)
		metadata := make(map[string]interface{})
		if len(bracket.Metadata) > 0 {
			if err := json.Unmarshal(bracket.Metadata, &metadata); err == nil {
				// Metadata parsed successfully
			}
		}

		// If we have data from AutoEnhance, include ALL their fields
		if aeBracket, exists := autoenhanceBrackets[bracket.BracketID]; exists {
			// Include all AutoEnhance fields
			response.IsUploaded = aeBracket.IsUploaded
			response.OrderID = aeBracket.OrderID
			response.Name = aeBracket.Name
			
			// Include image_id if available
			if aeBracket.ImageID != "" {
				response.ImageID = aeBracket.ImageID
			}
			
			// Merge AutoEnhance metadata with our database metadata
			// Our metadata (with group_id) takes precedence for grouping info
			if aeBracket.Metadata != nil {
				for k, v := range aeBracket.Metadata {
					// Only add AutoEnhance fields that we don't already have
					// This preserves our group_id from database
					if _, exists := metadata[k]; !exists {
						metadata[k] = v
					}
				}
			}
		}

		// Set merged metadata (includes group_id from our database)
		if len(metadata) > 0 {
			response.Metadata = metadata
		}

		bracketResponses[i] = response
	}

	c.JSON(http.StatusOK, models.BracketsResponse{Brackets: bracketResponses})
}

// DeleteBracket godoc
// @Summary     Delete an uploaded bracket
// @Description Deletes a bracket (uploaded raw image) from AutoEnhance AI. Note: Brackets are automatically cleaned up after successful processing.
// @Tags        brackets
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Param       bracket_id path string true "Bracket ID from AutoEnhance"
// @Success     200 {object} map[string]string "message"
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/brackets/{bracket_id} [delete]
func (h *FilesHandler) DeleteBracket(c *gin.Context) {
	if h.dbClient == nil || h.autoenhanceClient == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "services not available"})
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

	bracketID := c.Param("bracket_id")
	if bracketID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "bracket_id is required"})
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

	// Delete from AutoEnhance AI
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		return h.autoenhanceClient.DeleteBracket(bracketID)
	}, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to delete bracket from AutoEnhance",
			Message: err.Error(),
		})
		return
	}

	// Note: We keep the bracket record in our database for record-keeping
	// Only the AutoEnhance bracket is deleted

	c.JSON(http.StatusOK, gin.H{
		"message":    "Bracket deleted successfully from AutoEnhance",
		"bracket_id": bracketID,
	})
}
