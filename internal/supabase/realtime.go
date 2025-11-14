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

func (r *RealtimeClient) PublishProjectEvent(projectID uuid.UUID, event string, payload map[string]interface{}) error {
	channel := fmt.Sprintf("project:%s", projectID.String())
	return r.PublishEvent(channel, event, payload)
}

func (r *RealtimeClient) PublishUserEvent(userID uuid.UUID, event string, payload map[string]interface{}) error {
	channel := fmt.Sprintf("user:%s", userID.String())
	return r.PublishEvent(channel, event, payload)
}

// Event payloads
func UploadStartedPayload(projectID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "uploading",
		"file_count": fileCount,
	}
}

func UploadCompletedPayload(projectID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "uploaded",
		"file_count": fileCount,
	}
}

func ProcessingStartedPayload(projectID uuid.UUID, editID string) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "processing",
		"edit_id":    editID,
	}
}

func ProcessingProgressPayload(projectID uuid.UUID, progress int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "processing",
		"progress":   progress,
	}
}

func ProcessingCompletedPayload(projectID uuid.UUID, fileCount int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "completed",
		"progress":   100,
		"file_count": fileCount,
	}
}

func ProcessingFailedPayload(projectID uuid.UUID, errorMsg string) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID.String(),
		"status":     "failed",
		"error":      errorMsg,
	}
}

func DownloadReadyPayload(projectID uuid.UUID, storageURLs []string) map[string]interface{} {
	return map[string]interface{}{
		"project_id":   projectID.String(),
		"status":       "completed",
		"storage_urls": storageURLs,
	}
}
