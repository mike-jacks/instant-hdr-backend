package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID       `json:"id"`
	UserID           uuid.UUID       `json:"user_id"`
	ImagenProjectUUID string         `json:"imagen_project_uuid"`
	Status           string          `json:"status"`
	Progress         int             `json:"progress"`
	ProfileKey       sql.NullString  `json:"profile_key,omitempty"`
	EditID           sql.NullString  `json:"edit_id,omitempty"`
	Metadata         []byte          `json:"metadata"`
	ErrorMessage     sql.NullString  `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ProjectFile struct {
	ID            uuid.UUID      `json:"id"`
	ProjectID     uuid.UUID      `json:"project_id"`
	UserID        uuid.UUID      `json:"user_id"`
	Filename      string         `json:"filename"`
	ImagenFileID  sql.NullString `json:"imagen_file_id,omitempty"`
	StoragePath   string         `json:"storage_path"`
	StorageURL    string         `json:"storage_url"`
	FileSize      sql.NullInt64  `json:"file_size,omitempty"`
	MimeType      string         `json:"mime_type"`
	IsFinal       bool           `json:"is_final"`
	CreatedAt     time.Time      `json:"created_at"`
}

