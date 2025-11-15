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

type ImagesResponse struct {
	Images []ImageResponse `json:"images"`
}

type ImageResponse struct {
	ImageID            string                 `json:"image_id"`
	ImageName          string                 `json:"image_name"`
	Status             string                 `json:"status"`
	EnhanceType        string                 `json:"enhance_type,omitempty"`
	Downloaded         bool                   `json:"downloaded"`
	PreviewURL         string                 `json:"preview_url,omitempty"`          // Supabase URL for preview
	HighResURL         string                 `json:"high_res_url,omitempty"`         // Supabase URL for high-res
	PreviewDownloaded  bool                   `json:"preview_downloaded"`
	HighResDownloaded  bool                   `json:"high_res_downloaded"`
	ProcessingSettings map[string]interface{} `json:"processing_settings,omitempty"`
}

type DownloadImageRequest struct {
	// Preset quality options (recommended)
	Quality string `json:"quality"` // "thumbnail", "preview", "medium", "high", or "custom"
	
	// Custom options (used when quality="custom")
	MaxWidth *int     `json:"max_width,omitempty"` // Custom width in pixels
	Scale    *float64 `json:"scale,omitempty"`     // Scale factor (0.5 = 50%)
	Format   string   `json:"format,omitempty"`    // "jpeg", "png", "webp" (default: jpeg)
	
	// Watermark (optional, defaults to true)
	// true = FREE download with watermark
	// false = COSTS 1 CREDIT (unwatermarked)
	Watermark *bool `json:"watermark,omitempty"`
}

type DownloadImageResponse struct {
	ImageID    string `json:"image_id"`
	Quality    string `json:"quality"`
	URL        string `json:"url"`         // Supabase Storage URL
	FileSize   int64  `json:"file_size"`
	Watermark  bool   `json:"watermark"`   // true = FREE, false = COSTS 1 CREDIT
	Resolution string `json:"resolution,omitempty"` // e.g., "800px", "1920px", "full"
	Format     string `json:"format"`      // "jpeg", "png", "webp"
	CreditUsed bool   `json:"credit_used"` // true if this download cost a credit
	Message    string `json:"message"`
}
