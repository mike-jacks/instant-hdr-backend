package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Status       string
	Progress     int
	Metadata     json.RawMessage
	ErrorMessage sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OrderFile struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	UserID             uuid.UUID
	Filename           string
	AutoEnhanceImageID sql.NullString
	StoragePath        string
	StorageURL         string
	FileSize           sql.NullInt64
	MimeType           string
	IsFinal            bool
	CreatedAt          time.Time
}

type Bracket struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	BracketID  string
	ImageID    sql.NullString
	Filename   string
	UploadURL  sql.NullString
	IsUploaded bool
	Metadata   json.RawMessage
	CreatedAt  time.Time
}

