package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"instant-hdr-backend/internal/config"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/services"
)

type WebhookHandler struct {
	config         *config.Config
	storageService *services.StorageService
}

func NewWebhookHandler(cfg *config.Config, storageService *services.StorageService) *WebhookHandler {
	return &WebhookHandler{
		config:         cfg,
		storageService: storageService,
	}
}

// AutoEnhanceWebhookEvent represents AutoEnhance webhook event structure
type AutoEnhanceWebhookEvent struct {
	Event            string `json:"event"`              // "image_processed" or "webhook_updated"
	ImageID          string `json:"image_id,omitempty"` // The ID of the processed image
	Error            bool   `json:"error"`              // True if the image had an error
	OrderID          string `json:"order_id,omitempty"` // The ID of the order the image belongs to
	OrderIsProcessing bool  `json:"order_is_processing"` // True if order is processing, false if all images processed
}

// HandleWebhook godoc
// @Summary     AutoEnhance AI webhook endpoint
// @Description Receives webhook callbacks from AutoEnhance AI for processing status updates. Uses authentication token verification.
// @Tags        webhooks
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Authentication token (configured in AutoEnhance web app)"
// @Success     200 {object} map[string]string "status"
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Router      /webhooks/autoenhance [post]
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	if h.storageService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "storage service not available"})
		return
	}

	// Verify authentication token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "missing authorization token"})
		return
	}

	// Extract token (could be "Bearer <token>" or just "<token>")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	// Verify token matches configured webhook token
	if h.config.AutoEnhanceWebhookToken != "" && token != h.config.AutoEnhanceWebhookToken {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid authorization token"})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to read request body",
			Message: err.Error(),
		})
		return
	}

	// Parse JSON event
	var event AutoEnhanceWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse event",
			Message: err.Error(),
		})
		return
	}

	// Handle webhook_updated event (sent when webhook URL is configured)
	if event.Event == "webhook_updated" {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "webhook configured"})
		return
	}

	// Process image_processed events
	if event.Event == "image_processed" {
		if event.Error {
			// Image processing failed
			go h.storageService.HandleProcessingFailed(event.OrderID, "image processing failed")
		} else if !event.OrderIsProcessing {
			// All images in order are complete
			go h.storageService.HandleProcessingCompleted(event.OrderID, event.ImageID)
		}
		// If order_is_processing is true, more images are still being processed
		// We'll wait for the final event when order_is_processing is false
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
