package models

import "time"

type OrderResponse struct {
	ID                string                 `json:"order_id"`
	Name              string                 `json:"name,omitempty"` // Order name from AutoEnhance
	Status            string                 `json:"status"`
	Progress          int                    `json:"progress"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	// AutoEnhance data (when available)
	AutoEnhanceStatus string                   `json:"autoenhance_status,omitempty"`
	AutoEnhanceLastUpdatedAt *time.Time        `json:"autoenhance_last_updated_at,omitempty"` // Last update time from AutoEnhance
	TotalBrackets     int                      `json:"total_brackets,omitempty"`
	UploadedBrackets  int                      `json:"uploaded_brackets,omitempty"`
	TotalImages       int                      `json:"total_images,omitempty"`
	Images            []map[string]interface{} `json:"images,omitempty"`
	IsProcessing      bool                     `json:"is_processing,omitempty"`
	IsMerging         bool                     `json:"is_merging,omitempty"` // Indicates if brackets are currently being merged
	IsDeleted         bool                     `json:"is_deleted,omitempty"` // Indicates if order was deleted in AutoEnhance
}

type OrderListResponse struct {
	Orders []OrderSummary `json:"orders"`
}

type OrderSummary struct {
	ID        string    `json:"order_id"`
	Name      string    `json:"name,omitempty"` // Order name from AutoEnhance
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
	AutoEnhanceLastUpdatedAt *time.Time        `json:"autoenhance_last_updated_at,omitempty"` // Last update time from AutoEnhance
	TotalBrackets     int                      `json:"total_brackets,omitempty"`
	UploadedBrackets  int                      `json:"uploaded_brackets,omitempty"`
	TotalImages       int                      `json:"total_images,omitempty"`
	Images            []map[string]interface{} `json:"images,omitempty"`
	IsProcessing      bool                     `json:"is_processing,omitempty"`
	IsMerging          bool                     `json:"is_merging,omitempty"` // Indicates if brackets are currently being merged
	IsDeleted         bool                     `json:"is_deleted,omitempty"` // Indicates if order was deleted in AutoEnhance
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

// DownloadImageRequest defines the options for downloading a processed image
type DownloadImageRequest struct {
	// Quality preset - Options: "thumbnail" (400px), "preview" (800px), "medium" (1920px), "high" (full res), or "custom"
	// Default: "preview"
	Quality string `json:"quality" example:"preview"`
	
	// MaxWidth - Custom width in pixels (only used when quality="custom")
	// Default: null (not used unless quality="custom")
	MaxWidth *int `json:"max_width,omitempty"`
	
	// Scale - Scale factor (only used when quality="custom")
	// Default: null (not used unless quality="custom")
	// Example: 0.5 for 50% of original size
	Scale *float64 `json:"scale,omitempty"`
	
	// Format - Image format: "jpeg" (default), "png", or "webp"
	// Default: "jpeg"
	Format string `json:"format,omitempty" example:"jpeg"`
	
	// Watermark - Whether to include watermark. Defaults to true (FREE). Set to false to use 1 credit (unwatermarked)
	// Default: true (FREE - no credits used)
	Watermark *bool `json:"watermark,omitempty" example:"true"`
}

// DownloadImageResponse contains the result of downloading an image
type DownloadImageResponse struct {
	// ImageID from AutoEnhance
	ImageID string `json:"image_id" example:"img_abc123"`
	
	// Quality preset used
	Quality string `json:"quality" example:"preview"`
	
	// URL to access the image in Supabase Storage (publicly accessible)
	URL string `json:"url" example:"https://project.supabase.co/storage/v1/object/public/hdr-images/users/user123/orders/order456/img_abc123_preview.jpg"`
	
	// FileSize in bytes
	FileSize int64 `json:"file_size" example:"524288"`
	
	// Watermark indicates if watermark was applied (true = FREE, false = COSTS 1 CREDIT)
	Watermark bool `json:"watermark" example:"true"`
	
	// Resolution achieved (e.g., "400px", "800px", "1920px", "full")
	Resolution string `json:"resolution,omitempty" example:"800px"`
	
	// Format of the downloaded image
	Format string `json:"format" example:"jpeg"`
	
	// CreditUsed indicates if this download cost a credit
	CreditUsed bool `json:"credit_used" example:"false"`
	
	// Message with download details
	Message string `json:"message" example:"Image downloaded successfully (FREE with watermark) - Quality: preview, Resolution: 800px"`
}
