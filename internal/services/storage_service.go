package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type StorageService struct {
	autoenhanceClient *autoenhance.Client
	dbClient          *supabase.DatabaseClient
	storageClient     *supabase.StorageClient
	realtimeClient    *supabase.RealtimeClient
}

// GetRealtimeClient returns the realtime client for publishing events
func (s *StorageService) GetRealtimeClient() *supabase.RealtimeClient {
	return s.realtimeClient
}

func NewStorageService(
	autoenhanceClient *autoenhance.Client,
	dbClient *supabase.DatabaseClient,
	storageClient *supabase.StorageClient,
	realtimeClient *supabase.RealtimeClient,
) *StorageService {
	return &StorageService{
		autoenhanceClient: autoenhanceClient,
		dbClient:          dbClient,
		storageClient:     storageClient,
		realtimeClient:    realtimeClient,
	}
}

func (s *StorageService) HandleProcessingCompleted(autoenhanceOrderID, imageID string) {
	// Get order from database by order_id (AutoEnhance's order_id is our primary key)
	order, err := s.dbClient.GetOrderByAutoEnhanceOrderID(autoenhanceOrderID)
	if err != nil {
		// Order not found - log and return
		return
	}

	// Get order from AutoEnhance to get list of processed images
	autoenhanceOrder, err := s.autoenhanceClient.GetOrder(order.ID.String())
	if err != nil {
		s.dbClient.UpdateOrderError(order.ID, fmt.Sprintf("failed to get order from AutoEnhance: %v", err))
		return
	}

	// Sync AutoEnhance data to database (status, is_processing, total_images, etc.)
	var lastUpdated *time.Time
	if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
		lastUpdated = &autoenhanceOrder.LastUpdatedAt.Time
	}
	_ = s.dbClient.SyncAutoEnhanceOrderData(
		order.ID,
		autoenhanceOrder.Name,
		autoenhanceOrder.Status,
		autoenhanceOrder.IsProcessing,
		autoenhanceOrder.IsMerging,
		autoenhanceOrder.IsDeleted,
		int(autoenhanceOrder.TotalImages),
		lastUpdated,
	)

	if len(autoenhanceOrder.Images) == 0 {
		// No images yet - might still be processing
		return
	}

	// Download and store each processed image AS PREVIEW with watermark
	storageURLs := make([]string, 0)
	for _, image := range autoenhanceOrder.Images {
		// Skip if image has error or not completed
		if image.Status != "completed" || image.StatusReason != "" {
			continue
		}

		// Download PREVIEW image with watermark (FREE)
		watermark := true
		preview := true
		fileData, err := s.autoenhanceClient.DownloadEnhanced(image.ImageID, autoenhance.DownloadOptions{
			Format:    "jpeg",
			Preview:   &preview,   // Low-res preview
			Watermark: &watermark, // Free watermarked version
		})
		if err != nil {
			// Log error but continue with other images
			continue
		}

		// Generate filename with "preview" prefix
		filename := fmt.Sprintf("preview_%s_%s.jpg", image.ImageID[:8], time.Now().Format("20060102_150405"))

		// Upload to Supabase Storage
		storagePath, storageURL, err := s.storageClient.UploadFile(order.UserID, order.ID, filename, fileData)
		if err != nil {
			s.dbClient.UpdateOrderError(order.ID, fmt.Sprintf("failed to upload to storage: %v", err))
			continue
		}

		// Store file metadata in database (mark as preview, not final)
		file := &models.OrderFile{
			ID:                 uuid.New(),
			OrderID:            order.ID,
			UserID:             order.UserID,
			Filename:           filename,
			AutoEnhanceImageID: sql.NullString{String: image.ImageID, Valid: true},
			StoragePath:        storagePath,
			StorageURL:         storageURL,
			FileSize:           sql.NullInt64{Int64: int64(len(fileData)), Valid: true},
			MimeType:           "image/jpeg",
			IsFinal:            false, // This is a preview, not final high-res
			CreatedAt:          time.Now(),
		}

		if err := s.dbClient.CreateOrderFile(file); err != nil {
			// Log error but continue
		}

		storageURLs = append(storageURLs, storageURL)
	}

	if len(storageURLs) == 0 {
		// No images were successfully downloaded
		return
	}

	// Update order status to "previews_ready" instead of "completed"
	s.dbClient.UpdateOrderStatus(order.ID, "previews_ready", 100)

	// Publish download_ready event with preview URLs
	s.realtimeClient.PublishOrderEvent(order.ID, "download_ready",
		supabase.DownloadReadyPayload(order.ID, storageURLs))

	// Auto-cleanup: Delete brackets from AutoEnhance after successful processing
	// Brackets are no longer needed once images are processed
	go s.cleanupBrackets(order.ID.String())
}

// cleanupBrackets deletes all brackets for an order from AutoEnhance
// This is called after successful processing to save storage costs
func (s *StorageService) cleanupBrackets(orderID string) {
	// Get all brackets for the order
	brackets, err := s.autoenhanceClient.GetOrderBrackets(orderID)
	if err != nil {
		// Log error but don't fail - cleanup is best-effort
		return
	}

	// Delete each bracket from AutoEnhance
	for _, bracket := range brackets.Brackets {
		_ = s.autoenhanceClient.DeleteBracket(bracket.BracketID)
		// Errors are ignored - best-effort cleanup
		// Note: We keep brackets in our database for record-keeping
	}
}

func (s *StorageService) HandleProcessingFailed(autoenhanceOrderID, errorMsg string) {
	// Get order from database by order_id (AutoEnhance's order_id is our primary key)
	order, err := s.dbClient.GetOrderByAutoEnhanceOrderID(autoenhanceOrderID)
	if err != nil {
		// Order not found
		return
	}

	// Update order with error
	s.dbClient.UpdateOrderError(order.ID, errorMsg)

	// Sync AutoEnhance data to database (to get latest status, is_processing, etc.)
	autoenhanceOrder, err := s.autoenhanceClient.GetOrder(order.ID.String())
	if err == nil {
		var lastUpdated *time.Time
		if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
			lastUpdated = &autoenhanceOrder.LastUpdatedAt.Time
		}
		_ = s.dbClient.SyncAutoEnhanceOrderData(
			order.ID,
			autoenhanceOrder.Name,
			autoenhanceOrder.Status,
			autoenhanceOrder.IsProcessing,
			autoenhanceOrder.IsMerging,
			autoenhanceOrder.IsDeleted,
			int(autoenhanceOrder.TotalImages),
			lastUpdated,
		)
	}

	// Publish failed event
	s.realtimeClient.PublishOrderEvent(order.ID, "processing_failed",
		supabase.ProcessingFailedPayload(order.ID, errorMsg))
}
