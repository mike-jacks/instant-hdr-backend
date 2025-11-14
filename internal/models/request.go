package models

type CreateProjectRequest struct {
	// Optional metadata to store with project
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ProcessRequest struct {
	// ProfileKey is an integer that identifies the editing profile to use.
	// Get available profiles from GET /profiles endpoint.
	// Can be sent as string (e.g., "123") and will be converted to integer.
	ProfileKey string   `json:"profile_key,omitempty" example:"123"`
	HDRMerge   bool     `json:"hdr_merge" example:"true"`
	AITools    []string `json:"ai_tools,omitempty"` // Deprecated: not used in current Imagen API
	// Optional metadata to store with the processing request
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
