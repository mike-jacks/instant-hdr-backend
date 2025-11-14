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
	// Get order from database by autoenhance_order_id
	order, err := s.dbClient.GetOrderByAutoEnhanceOrderID(autoenhanceOrderID)
	if err != nil {
		// Order not found - log and return
		return
	}

	// Get order from AutoEnhance to get list of processed images
	autoenhanceOrder, err := s.autoenhanceClient.GetOrder(autoenhanceOrderID)
	if err != nil {
		s.dbClient.UpdateOrderError(order.ID, fmt.Sprintf("failed to get order from AutoEnhance: %v", err))
		return
	}

	if len(autoenhanceOrder.Images) == 0 {
		// No images yet - might still be processing
		return
	}

	// Download and store each processed image
	storageURLs := make([]string, 0)
	for _, image := range autoenhanceOrder.Images {
		// Skip if image has error or not completed
		if image.Status != "completed" || image.StatusReason != "" {
			continue
		}

		// Download enhanced image
		fileData, err := s.autoenhanceClient.DownloadEnhanced(image.ImageID, autoenhance.DownloadOptions{
			Format: "jpeg",
		})
		if err != nil {
			// Log error but continue with other images
			continue
		}

		// Generate filename
		filename := fmt.Sprintf("enhanced_%s_%s.jpg", image.ImageID[:8], time.Now().Format("20060102_150405"))

		// Upload to Supabase Storage
		storagePath, storageURL, err := s.storageClient.UploadFile(order.UserID, order.ID, filename, fileData)
		if err != nil {
			s.dbClient.UpdateOrderError(order.ID, fmt.Sprintf("failed to upload to storage: %v", err))
			continue
		}

		// Store file metadata in database
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
			IsFinal:            true,
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

	// Update order status
	s.dbClient.UpdateOrderStatus(order.ID, "completed", 100)

	// Publish download_ready event
	s.realtimeClient.PublishOrderEvent(order.ID, "download_ready",
		supabase.DownloadReadyPayload(order.ID, storageURLs))
}

func (s *StorageService) HandleProcessingFailed(autoenhanceOrderID, errorMsg string) {
	order, err := s.dbClient.GetOrderByAutoEnhanceOrderID(autoenhanceOrderID)
	if err != nil {
		// Order not found
		return
	}

	// Update order with error
	s.dbClient.UpdateOrderError(order.ID, errorMsg)

	// Publish failed event
	s.realtimeClient.PublishOrderEvent(order.ID, "processing_failed",
		supabase.ProcessingFailedPayload(order.ID, errorMsg))
}
