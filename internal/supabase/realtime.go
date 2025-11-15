package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type RealtimeClient struct {
	client          *supabase.Client
	supabaseURL     string
	publishableKey  string
	httpClient      *http.Client
}

func NewRealtimeClient(client *supabase.Client, supabaseURL, publishableKey string) *RealtimeClient {
	return &RealtimeClient{
		client:         client,
		supabaseURL:     supabaseURL,
		publishableKey: publishableKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PublishEvent publishes a custom event to a Supabase Realtime channel using the REST API
// This allows frontend to receive events via broadcast listeners
// Uses publishable key for authentication (can also use service role key)
// Based on: https://supabase.com/docs/guides/realtime/broadcast
func (r *RealtimeClient) PublishEvent(channel string, event string, payload map[string]interface{}) error {
	if r.publishableKey == "" {
		// If no publishable key, skip publishing (graceful degradation)
		return nil
	}

	// Add timestamp to payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["timestamp"] = time.Now().Format(time.RFC3339)

	// Prepare request body according to Supabase API format
	// Format: { "messages": [{ "topic": "...", "event": "...", "payload": {...} }] }
	requestBody := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"topic":   channel,
				"event":   event,
				"payload": payload,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Call Supabase Realtime REST API
	// Endpoint: POST /realtime/v1/api/broadcast
	// Docs: https://supabase.com/docs/guides/realtime/broadcast
	url := fmt.Sprintf("%s/realtime/v1/api/broadcast", r.supabaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", r.publishableKey)
	// Note: Authorization header is optional, apikey header is sufficient
	// But including both for compatibility

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Read response body for error details
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes[:n]))
	}

	return nil
}

func (r *RealtimeClient) PublishOrderEvent(orderID uuid.UUID, event string, payload map[string]interface{}) error {
	channel := fmt.Sprintf("order:%s", orderID.String())
	return r.PublishEvent(channel, event, payload)
}

// Deprecated: Use PublishOrderEvent instead
func (r *RealtimeClient) PublishProjectEvent(projectID uuid.UUID, event string, payload map[string]interface{}) error {
	return r.PublishOrderEvent(projectID, event, payload)
}

func (r *RealtimeClient) PublishUserEvent(userID uuid.UUID, event string, payload map[string]interface{}) error {
	channel := fmt.Sprintf("user:%s", userID.String())
	return r.PublishEvent(channel, event, payload)
}

// Event payloads
func UploadStartedPayload(orderID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"order_id":  orderID.String(),
		"status":    "uploading",
		"file_count": fileCount,
	}
}

func UploadCompletedPayload(orderID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"order_id":  orderID.String(),
		"status":    "uploaded",
		"file_count": fileCount,
	}
}

func ProcessingStartedPayload(orderID uuid.UUID, editID string) map[string]interface{} {
	return map[string]interface{}{
		"order_id": orderID.String(),
		"status":   "processing",
	}
}

func ProcessingProgressPayload(orderID uuid.UUID, progress int) map[string]interface{} {
	return map[string]interface{}{
		"order_id": orderID.String(),
		"status":   "processing",
		"progress": progress,
	}
}

func ProcessingCompletedPayload(orderID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"order_id":  orderID.String(),
		"status":    "completed",
		"progress":  100,
		"file_count": fileCount,
	}
}

func ProcessingFailedPayload(orderID uuid.UUID, errorMsg string) map[string]interface{} {
	return map[string]interface{}{
		"order_id": orderID.String(),
		"status":   "failed",
		"error":    errorMsg,
	}
}

func DownloadReadyPayload(orderID uuid.UUID, storageURLs []string) map[string]interface{} {
	return map[string]interface{}{
		"order_id":     orderID.String(),
		"status":       "previews_ready",
		"storage_urls": storageURLs,
	}
}

// WebhookEventPayload creates a payload from AutoEnhance webhook data
func WebhookEventPayload(orderID, imageID string, error bool, orderIsProcessing bool) map[string]interface{} {
	return map[string]interface{}{
		"order_id":            orderID,
		"image_id":            imageID,
		"error":               error,
		"order_is_processing": orderIsProcessing,
	}
}
