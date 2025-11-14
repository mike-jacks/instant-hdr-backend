package models

import "time"

type OrderResponse struct {
	ID                string                 `json:"order_id"`
	Status            string                 `json:"status"`
	Progress          int                    `json:"progress"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	// AutoEnhance data (when available)
	AutoEnhanceStatus string                   `json:"autoenhance_status,omitempty"`
	TotalBrackets     int                      `json:"total_brackets,omitempty"`
	UploadedBrackets  int                      `json:"uploaded_brackets,omitempty"`
	TotalImages       int                      `json:"total_images,omitempty"`
	Images            []map[string]interface{} `json:"images,omitempty"`
	IsProcessing      bool                     `json:"is_processing,omitempty"`
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
	OrderID string            `json:"order_id"`
	Files   []FileInfo        `json:"files"`
	Status  string            `json:"status"`
	Errors  []UploadErrorInfo `json:"errors,omitempty"`
}

type UploadErrorInfo struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
	Stage    string `json:"stage"` // "create_bracket", "upload", "verify", "database"
}

type FileInfo struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type ProcessResponse struct {
	OrderID          string                 `json:"order_id"`
	Status           string                 `json:"status"`
	Message          string                 `json:"message,omitempty"`
	ProcessingParams map[string]interface{} `json:"processing_params,omitempty"`
}

type StatusResponse struct {
	OrderID           string                   `json:"order_id"`
	Status            string                   `json:"status"`
	Progress          int                      `json:"progress"`
	UpdatedAt         time.Time                `json:"updated_at"`
	// AutoEnhance data
	AutoEnhanceStatus string                   `json:"autoenhance_status,omitempty"`
	TotalBrackets     int                      `json:"total_brackets,omitempty"`
	UploadedBrackets  int                      `json:"uploaded_brackets,omitempty"`
	TotalImages       int                      `json:"total_images,omitempty"`
	Images            []map[string]interface{} `json:"images,omitempty"`
	IsProcessing      bool                     `json:"is_processing,omitempty"`
}

type FilesResponse struct {
	Files []FileResponse `json:"files"`
}

type BracketsResponse struct {
	Brackets []BracketResponse `json:"brackets"`
}

type BracketResponse struct {
	ID         string                 `json:"id"`
	BracketID  string                 `json:"bracket_id"`
	Filename   string                 `json:"filename"`
	IsUploaded bool                   `json:"is_uploaded"`
	CreatedAt  time.Time              `json:"created_at"`
	ImageID    string                 `json:"image_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
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
