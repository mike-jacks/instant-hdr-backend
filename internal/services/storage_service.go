package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type StorageService struct {
	imagenClient   *imagen.Client
	dbClient       *supabase.DatabaseClient
	storageClient  *supabase.StorageClient
	realtimeClient *supabase.RealtimeClient
}

func NewStorageService(
	imagenClient *imagen.Client,
	dbClient *supabase.DatabaseClient,
	storageClient *supabase.StorageClient,
	realtimeClient *supabase.RealtimeClient,
) *StorageService {
	return &StorageService{
		imagenClient:   imagenClient,
		dbClient:       dbClient,
		storageClient:  storageClient,
		realtimeClient: realtimeClient,
	}
}

func (s *StorageService) HandleProcessingCompleted(imagenProjectUUID, eventID string) {
	// Find project by imagen_project_uuid
	// Note: This requires a new database method or we store the mapping
	// For now, we'll need to query by imagen_project_uuid

	// Get project from database (we need to add this method)
	// For now, let's assume we can get it somehow
	// This is a simplified version - in production, you'd query the database

	// Export from Imagen (submits export request)
	err := s.imagenClient.Export(imagenProjectUUID)
	if err != nil {
		// Handle error - update project status
		return
	}

	// Wait for export to complete and get download links
	// In production, you'd poll GetExportStatus until completed
	// For now, we'll try to get download links (they may not be ready yet)
	downloadLinks, err := s.imagenClient.GetExportDownloadLinks(imagenProjectUUID)
	if err != nil {
		// Export may not be ready yet - this is expected
		// In production, implement polling with GetExportStatus
		return
	}

	if len(downloadLinks) == 0 {
		// No files available yet
		return
	}

	// Download the first file (or all files - adjust as needed)
	fileData, err := s.imagenClient.DownloadFile(downloadLinks[0].DownloadLink)
	if err != nil {
		// Handle error
		return
	}

	// For now, we need the project ID and user ID
	// This is a limitation - we need to store the mapping or query differently
	// Let's create a helper method to get project by imagen UUID
	project, userID, projectID := s.getProjectByImagenUUID(imagenProjectUUID)
	if project == nil {
		return
	}

	// Generate filename
	filename := fmt.Sprintf("merged_hdr_%s.jpg", time.Now().Format("20060102_150405"))

	// Upload to Supabase Storage
	storagePath, storageURL, err := s.storageClient.UploadFile(userID, projectID, filename, fileData)
	if err != nil {
		s.dbClient.UpdateProjectError(projectID, fmt.Sprintf("failed to upload to storage: %v", err))
		return
	}

	// Store file metadata in database
	file := &models.ProjectFile{
		ID:          uuid.New(),
		ProjectID:   projectID,
		UserID:      userID,
		Filename:    filename,
		StoragePath: storagePath,
		StorageURL:  storageURL,
		FileSize:    sql.NullInt64{Int64: int64(len(fileData)), Valid: true},
		MimeType:    "image/jpeg",
		IsFinal:     true,
		CreatedAt:   time.Now(),
	}

	if err := s.dbClient.CreateProjectFile(file); err != nil {
		// Log error but continue
	}

	// Update project status
	s.dbClient.UpdateProjectStatus(projectID, "completed", 100)

	// Publish download_ready event
	s.realtimeClient.PublishProjectEvent(projectID, "download_ready",
		supabase.DownloadReadyPayload(projectID, []string{storageURL}))
}

func (s *StorageService) HandleProcessingFailed(imagenProjectUUID, errorMsg string) {
	project, _, projectID := s.getProjectByImagenUUID(imagenProjectUUID)
	if project == nil {
		return
	}

	// Update project with error
	s.dbClient.UpdateProjectError(projectID, errorMsg)

	// Publish failed event
	s.realtimeClient.PublishProjectEvent(projectID, "processing_failed",
		supabase.ProcessingFailedPayload(projectID, errorMsg))
}

// Helper method to get project by imagen UUID
func (s *StorageService) getProjectByImagenUUID(imagenUUID string) (*models.Project, uuid.UUID, uuid.UUID) {
	project, err := s.dbClient.GetProjectByImagenUUID(imagenUUID)
	if err != nil {
		return nil, uuid.Nil, uuid.Nil
	}
	return project, project.UserID, project.ID
}
