package supabase

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type RealtimeClient struct {
	client *supabase.Client
}

func NewRealtimeClient(client *supabase.Client) *RealtimeClient {
	return &RealtimeClient{
		client: client,
	}
}

func (r *RealtimeClient) PublishEvent(channel string, event string, payload map[string]interface{}) error {
	// Note: Supabase Go client doesn't have direct Realtime publish
	// We'll use database updates which trigger Realtime automatically
	// For explicit events, we can use the Realtime REST API if needed

	// For now, database updates will trigger Realtime automatically
	// This is a placeholder for future explicit event publishing
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
		"order_id":    orderID.String(),
		"status":      "completed",
		"storage_urls": storageURLs,
	}
}
