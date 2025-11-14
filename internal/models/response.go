package models

import "time"

type ProjectResponse struct {
	ID               string                 `json:"project_id"`
	ImagenProjectUUID string                `json:"imagen_project_uuid"`
	Status           string                 `json:"status"`
	Progress         int                    `json:"progress"`
	ProfileKey       string                 `json:"profile_key,omitempty"`
	EditID           string                 `json:"edit_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type ProjectListResponse struct {
	Projects []ProjectSummary `json:"projects"`
}

type ProjectSummary struct {
	ID               string    `json:"project_id"`
	Status           string    `json:"status"`
	Progress         int       `json:"progress"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UploadResponse struct {
	ProjectID string      `json:"project_id"`
	Files     []FileInfo  `json:"files"`
	Status    string      `json:"status"`
}

type FileInfo struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type ProcessResponse struct {
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
	EditID    string `json:"edit_id"`
}

type StatusResponse struct {
	ProjectID string    `json:"project_id"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FilesResponse struct {
	Files []FileResponse `json:"files"`
}

type FileResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	StorageURL  string    `json:"storage_url"`
	FileSize    int64     `json:"file_size"`
	MimeType    string    `json:"mime_type"`
	IsFinal     bool      `json:"is_final"`
	CreatedAt   time.Time `json:"created_at"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

