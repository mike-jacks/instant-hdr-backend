package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type RealtimeClient struct {
	client         *supabase.Client
	supabaseURL    string
	serviceRoleKey string
	httpClient     *http.Client
}

func NewRealtimeClient(client *supabase.Client, supabaseURL, serviceRoleKey string) *RealtimeClient {
	return &RealtimeClient{
		client:         client,
		supabaseURL:    supabaseURL,
		serviceRoleKey: serviceRoleKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PublishEvent publishes a custom event to a Supabase Realtime channel using the REST API
// This allows frontend to receive events via broadcast listeners
// Uses service role key for authentication (bypasses RLS)
// Based on: https://supabase.com/docs/guides/realtime/broadcast
func (r *RealtimeClient) PublishEvent(channel string, event string, payload map[string]interface{}) error {
	// Use service role key for server-side publishing
	if r.serviceRoleKey == "" {
		// If no service role key available, skip publishing (graceful degradation)
		log.Printf("[Realtime] Skipping publish: service role key not configured (channel: %s, event: %s)", channel, event)
		return nil
	}

	if r.supabaseURL == "" {
		return fmt.Errorf("supabase URL is not configured")
	}

	log.Printf("[Realtime] Publishing event: channel=%s, event=%s", channel, event)

	// Add timestamp to payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["timestamp"] = time.Now().Format(time.RFC3339)

	// Prepare request body according to Supabase API format
	// Format: { "messages": [{ "topic": "...", "event": "...", "payload": {...} }] }
	// Docs: https://supabase.com/docs/guides/realtime/broadcast
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
	// According to docs, only 'apikey' header is needed (publishable/anon key)
	// Example from docs:
	//   curl -v \
	//   -H 'apikey: <SUPABASE_TOKEN>' \
	//   -H 'Content-Type: application/json' \
	//   --data-raw '{ "messages": [{ "topic": "test", "event": "event", "payload": { "test": "test" } }] }' \
	//   'https://<PROJECT_REF>.supabase.co/realtime/v1/api/broadcast'
	url := fmt.Sprintf("%s/realtime/v1/api/broadcast", r.supabaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers as per Supabase docs
	// Use service role key for server-side publishing (bypasses RLS)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", r.serviceRoleKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("[Realtime] Failed to send request: channel=%s, event=%s, error=%v", channel, event, err)
		return fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes := make([]byte, 2048)
	n, _ := resp.Body.Read(bodyBytes)
	responseBody := string(bodyBytes[:n])

	// Accept 200 (OK), 201 (Created), and 202 (Accepted) as success status codes
	// 202 is commonly returned for asynchronous operations
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		log.Printf("[Realtime] Broadcast failed: channel=%s, event=%s, status=%d, body=%s",
			channel, event, resp.StatusCode, responseBody)
		return fmt.Errorf("realtime broadcast failed: status %d, channel: %s, event: %s, body: %s. "+
			"Verify: 1) SUPABASE_URL is correct (should be https://<project-ref>.supabase.co), "+
			"2) SUPABASE_SERVICE_ROLE_KEY is correct (should start with 'eyJ...' or 'sb_...'), "+
			"3) Service role key has proper permissions",
			resp.StatusCode, channel, event, responseBody)
	}

	log.Printf("[Realtime] Successfully published event: channel=%s, event=%s, status=%d",
		channel, event, resp.StatusCode)
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
		"order_id":   orderID.String(),
		"status":     "uploading",
		"file_count": fileCount,
	}
}

func UploadCompletedPayload(orderID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"order_id":   orderID.String(),
		"status":     "uploaded",
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
		"order_id":   orderID.String(),
		"status":     "completed",
		"progress":   100,
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
