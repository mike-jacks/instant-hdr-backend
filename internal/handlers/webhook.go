package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/config"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/services"
	"instant-hdr-backend/internal/supabase"
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

	// Log all headers for debugging
	log.Printf("[Webhook] Received webhook request from %s", c.ClientIP())
	log.Printf("[Webhook] Headers: %v", c.Request.Header)

	// Verify authentication token (only if webhook token is configured)
	if h.config.AutoEnhanceWebhookToken != "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("[Webhook] Missing Authorization header (webhook token is configured)")
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "missing authorization token"})
			return
		}

		// Extract token (could be "Bearer <token>" or just "<token>")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		// Verify token matches configured webhook token
		if token != h.config.AutoEnhanceWebhookToken {
			log.Printf("[Webhook] Invalid token: received='%s' (length: %d), expected='%s' (length: %d)",
				token, len(token), h.config.AutoEnhanceWebhookToken, len(h.config.AutoEnhanceWebhookToken))
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid authorization token"})
			return
		}
		log.Printf("[Webhook] Token validated successfully")
	} else {
		log.Printf("[Webhook] Warning: AUTOENHANCE_WEBHOOK_TOKEN not configured, skipping authentication")
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

	// Handle empty body (could be a test/verification request from AutoEnhance)
	if len(body) == 0 {
		// AutoEnhance may send empty body for webhook verification/test
		// Return success to acknowledge the webhook is configured
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "webhook endpoint is active and ready to receive events",
		})
		return
	}

	// Parse JSON event
	var event AutoEnhanceWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		// Log the raw body for debugging
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "... (truncated)"
		}
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse event",
			Message: fmt.Sprintf("invalid JSON: %v. Received body: %s", err, bodyStr),
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
		// Parse order ID to UUID for publishing
		orderID, err := uuid.Parse(event.OrderID)
		if err == nil && h.storageService != nil {
			// Publish EVERY webhook event to frontend immediately
			// Frontend can track individual image processing progress
			webhookPayload := supabase.WebhookEventPayload(
				event.OrderID,
				event.ImageID,
				event.Error,
				event.OrderIsProcessing,
			)
			
			// Publish to realtime channel (async, don't block webhook response)
			go func() {
				_ = h.storageService.GetRealtimeClient().PublishOrderEvent(
					orderID,
					"webhook_image_processed",
					webhookPayload,
				)
			}()
		}

		// Handle business logic based on webhook data
		if event.Error {
			// Image processing failed
			go h.storageService.HandleProcessingFailed(event.OrderID, "image processing failed")
		} else if !event.OrderIsProcessing {
			// All images in order are complete
			go h.storageService.HandleProcessingCompleted(event.OrderID, event.ImageID)
		}
		// If order_is_processing is true, more images are still being processed
		// Frontend will receive individual events for each image
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
