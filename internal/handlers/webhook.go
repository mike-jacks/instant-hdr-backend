package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"

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

// WebhookEvent represents Imagen webhook event structure
type WebhookEvent struct {
	XMLName xml.Name `xml:"event"`
	ID      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr"`
	Project string   `xml:"project"`
	Status  string   `xml:"status"`
	Message string   `xml:"message,omitempty"`
}

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	// Handle challenge parameter for webhook setup
	challenge := c.Query("challenge")
	if challenge != "" {
		c.String(http.StatusOK, challenge)
		return
	}

	if h.storageService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "storage service not available"})
		return
	}

	// Verify HMAC signature
	signature := c.GetHeader("X-Imagen-Webhook")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "missing signature"})
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

	// Verify signature
	if !h.verifySignature(signature, c.Request.URL.String(), body) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid signature"})
		return
	}

	// Parse XML event
	var event WebhookEvent
	if err := xml.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse event",
			Message: err.Error(),
		})
		return
	}

	// Process event based on type
	switch event.Type {
	case "processing_completed":
		// Trigger automatic storage
		go h.storageService.HandleProcessingCompleted(event.Project, event.ID)
	case "processing_failed":
		// Update project status
		go h.storageService.HandleProcessingFailed(event.Project, event.Message)
	case "processing_progress":
		// Update progress (if provided in message)
		// This would need parsing the message for progress percentage
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) verifySignature(signature, url string, body []byte) bool {
	// Create HMAC hash
	mac := hmac.New(sha256.New, []byte(h.config.ImagenWebhookSecret))
	mac.Write([]byte(url))
	mac.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

