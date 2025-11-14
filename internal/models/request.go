package models

type CreateProjectRequest struct {
	// Optional metadata to store with project
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ProcessRequest struct {
	ProfileKey string   `json:"profile_key,omitempty"`
	HDRMerge   bool     `json:"hdr_merge"`
	AITools    []string `json:"ai_tools,omitempty"`
	// Add other Imagen API options as needed
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

