package models

import "time"

type OrderResponse struct {
	ID           string                 `json:"order_id"`
	Status       string                 `json:"status"`
	Progress     int                    `json:"progress"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type OrderListResponse struct {
	Orders []OrderSummary `json:"orders"`
}

type OrderSummary struct {
	ID        string    `json:"order_id"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UploadResponse struct {
	OrderID string     `json:"order_id"`
	Files   []FileInfo `json:"files"`
	Status  string     `json:"status"`
	Errors  []string   `json:"errors,omitempty"`
}

type FileInfo struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type ProcessResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type StatusResponse struct {
	OrderID  string    `json:"order_id"`
	Status   string    `json:"status"`
	Progress int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FilesResponse struct {
	Files []FileResponse `json:"files"`
}

type BracketsResponse struct {
	Brackets []BracketResponse `json:"brackets"`
}

type BracketResponse struct {
	ID         string    `json:"id"`
	BracketID  string    `json:"bracket_id"`
	Filename   string    `json:"filename"`
	IsUploaded bool      `json:"is_uploaded"`
	CreatedAt  time.Time `json:"created_at"`
}

type FileResponse struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	StorageURL string    `json:"storage_url"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	IsFinal    bool      `json:"is_final"`
	CreatedAt  time.Time `json:"created_at"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
