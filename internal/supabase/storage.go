package supabase

import (
	"bytes"
	"fmt"

	"github.com/google/uuid"
	storage "github.com/supabase-community/storage-go"
)

type StorageClient struct {
	client  *storage.Client
	bucket  string
	baseURL string
}

func NewStorageClient(supabaseURL, serviceRoleKey, bucket string) (*StorageClient, error) {
	// Ensure URL doesn't have trailing slash
	baseURL := supabaseURL
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	client := storage.NewClient(baseURL+"/storage/v1", serviceRoleKey, nil)
	
	return &StorageClient{
		client:  client,
		bucket:  bucket,
		baseURL: baseURL,
	}, nil
}

func (s *StorageClient) UploadFile(userID, projectID uuid.UUID, filename string, data []byte) (string, string, error) {
	// Create storage path: users/{user_id}/projects/{project_id}/{filename}
	storagePath := fmt.Sprintf("users/%s/projects/%s/%s", userID.String(), projectID.String(), filename)
	
	// Upload file
	contentType := "image/jpeg"
	upsert := true
	_, err := s.client.UploadFile(s.bucket, storagePath, bytes.NewReader(data), storage.FileOptions{
		ContentType: &contentType,
		Upsert:      &upsert,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", 
		s.baseURL, s.bucket, storagePath)
	
	return storagePath, publicURL, nil
}

func (s *StorageClient) GetPublicURL(storagePath string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
		s.baseURL, s.bucket, storagePath)
}

func (s *StorageClient) DeleteFile(storagePath string) error {
	_, err := s.client.RemoveFile(s.bucket, []string{storagePath})
	return err
}

func (s *StorageClient) DeleteProjectFiles(userID, projectID uuid.UUID) error {
	prefix := fmt.Sprintf("users/%s/projects/%s/", userID.String(), projectID.String())
	
	// List files with prefix
	files, err := s.client.ListFiles(s.bucket, prefix, storage.FileSearchOptions{
		Limit: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Delete all files
	if len(files) > 0 {
		filePaths := make([]string, len(files))
		for i, file := range files {
			filePaths[i] = file.Name
		}
		_, err = s.client.RemoveFile(s.bucket, filePaths)
		if err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
	}

	return nil
}

func (s *StorageClient) DownloadFile(storagePath string) ([]byte, error) {
	data, err := s.client.DownloadFile(s.bucket, storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	
	return data, nil
}

