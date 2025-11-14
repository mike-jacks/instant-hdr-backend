package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	ImagenProjectUUID string
	Status            string
	Progress          int
	ProfileKey        sql.NullString
	EditID            sql.NullString
	Metadata          json.RawMessage
	ErrorMessage      sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ProjectFile struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	UserID       uuid.UUID
	Filename     string
	ImagenFileID sql.NullString
	StoragePath  string
	StorageURL   string
	FileSize     sql.NullInt64
	MimeType     string
	IsFinal      bool
	CreatedAt    time.Time
}
