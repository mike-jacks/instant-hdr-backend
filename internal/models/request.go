package models

type CreateOrderRequest struct {
	// Optional metadata to store with order
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ProcessRequest struct {
	// EnhanceType specifies the type of enhancement to apply
	// Options: "property", "property_usa", "warm", "neutral", "modern"
	// Default: "property" for real estate photography
	EnhanceType      string `json:"enhance_type,omitempty" example:"property"`
	SkyReplacement   *bool  `json:"sky_replacement,omitempty" example:"true"`
	WindowPullType   string `json:"window_pull_type,omitempty" example:"ONLY_WINDOWS"` // "NONE", "ONLY_WINDOWS", "WINDOWS_WITH_SKIES"
	VerticalCorrection *bool `json:"vertical_correction,omitempty" example:"true"`
	LensCorrection   *bool  `json:"lens_correction,omitempty" example:"true"`
	Upscale          *bool  `json:"upscale,omitempty" example:"false"`
	Privacy          *bool  `json:"privacy,omitempty" example:"false"`
	// Optional metadata to store with the processing request
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
