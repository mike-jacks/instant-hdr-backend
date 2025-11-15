package models

type CreateOrderRequest struct {
	// Order name/description (e.g., "123 Main St - Living Room")
	// If not provided, defaults to "Order"
	Name string `json:"name,omitempty" example:"Property Shoot - 123 Main St"`
}

type ProcessRequest struct {
	// EnhanceType specifies the type of enhancement to apply to the image.
	// Options: "property", "property_usa", "warm", "neutral", "modern"
	// - "property" (DEFAULT): Best for real estate photography - balanced enhancement
	// - "property_usa": USA-specific real estate enhancement (for AI version < 4.0)
	// - "warm": Warm color grading for cozy, inviting feel (AI version >= 4.0)
	// - "neutral": Neutral color grading for natural look (AI version >= 4.0)
	// - "modern": Modern, contemporary enhancement style (AI version >= 4.0)
	// Default: "property"
	EnhanceType string `json:"enhance_type,omitempty" example:"property" enums:"property,property_usa,warm,neutral,modern"`

	// SkyReplacement enables AI-powered sky replacement with realistic clouds.
	// When enabled, dull or overcast skies are replaced with attractive blue skies.
	// Works best with outdoor property photos.
	// Default: true (recommended for real estate)
	SkyReplacement *bool `json:"sky_replacement,omitempty" example:"true"`

	// CloudType specifies what type of clouds to add when sky replacement is enabled.
	// Options: "CLEAR", "LOW_CLOUD", "HIGH_CLOUD"
	// - "CLEAR": Clear blue sky with minimal clouds
	// - "LOW_CLOUD": Low-altitude clouds for subtle effect
	// - "HIGH_CLOUD": High-altitude clouds for more dramatic skies
	// Default: null (AutoEnhance chooses automatically based on scene)
	CloudType string `json:"cloud_type,omitempty" enums:"CLEAR,LOW_CLOUD,HIGH_CLOUD"`

	// WindowPullType controls how window views are enhanced.
	// Options: "NONE", "ONLY_WINDOWS", "WINDOWS_WITH_SKIES"
	// - "NONE": No window enhancement (keep original window views)
	// - "ONLY_WINDOWS": Enhance window views only (bring out detail)
	// - "WINDOWS_WITH_SKIES" (DEFAULT): Enhance windows AND replace exterior skies visible through windows (requires AI version >= 5.2)
	// Default: "WINDOWS_WITH_SKIES" (recommended for best results)
	WindowPullType string `json:"window_pull_type,omitempty" example:"WINDOWS_WITH_SKIES" enums:"NONE,ONLY_WINDOWS,WINDOWS_WITH_SKIES"`

	// VerticalCorrection automatically corrects vertical perspective distortion.
	// Straightens walls and vertical lines that appear tilted due to camera angle.
	// Essential for professional real estate photography.
	// Default: true (highly recommended)
	VerticalCorrection *bool `json:"vertical_correction,omitempty" example:"true"`

	// LensCorrection removes lens distortion (barrel/pincushion effect).
	// Corrects curved lines caused by wide-angle lenses commonly used in real estate.
	// Recommended for all real estate photos.
	// Default: true
	LensCorrection *bool `json:"lens_correction,omitempty" example:"true"`

	// Upscale increases image resolution using AI upscaling technology.
	// Doubles the resolution while maintaining quality.
	// Warning: Significantly increases processing time and final file size.
	// Default: false
	Upscale *bool `json:"upscale,omitempty" example:"false"`

	// Privacy blurs faces and license plates for privacy compliance.
	// Useful for public listings where privacy protection is required.
	// Default: false
	Privacy *bool `json:"privacy,omitempty" example:"false"`

	// AIVersion specifies the AI model version to use for processing.
	// Examples: "4.0", "5.2", "5.x"
	// Versions ending in .x (e.g., "5.x") will automatically use the latest minor version.
	// Default: Latest stable version (automatically selected by AutoEnhance)
	AIVersion string `json:"ai_version,omitempty"`

	// BracketGrouping specifies how uploaded brackets are organized into HDR images.
	// Options: "by_upload_group", "auto", "all", "individual", or custom array
	// - "by_upload_group" (RECOMMENDED): Groups brackets by group_id assigned during upload
	// - "auto": Groups brackets sequentially by sets (e.g., every 3 brackets = 1 HDR)
	// - "all": Merges ALL brackets into ONE HDR image (maximum dynamic range)
	// - "individual": Each bracket becomes a separate image (no HDR merging)
	// - Custom array: [[id1,id2,id3],[id4,id5]] - Specify exact bracket groupings by bracket_id
	// Default: "by_upload_group"
	BracketGrouping interface{} `json:"bracket_grouping,omitempty" swaggertype:"string" example:"by_upload_group"`

	// BracketsPerImage specifies how many consecutive brackets to group into one HDR image.
	// Only used when bracket_grouping is "auto". This tells the system to group brackets sequentially.
	// IGNORED if bracket_grouping is "by_upload_group" (the default).
	//
	// Example with 6 brackets and brackets_per_image=3:
	//   - Brackets [1,2,3] → HDR Image #1
	//   - Brackets [4,5,6] → HDR Image #2
	//
	// Common values:
	// - 3 (DEFAULT when using "auto"): Standard 3-exposure HDR (underexposed, normal, overexposed)
	//   Best for: Most real estate shots, typical bracketing workflows
	// - 5: 5-exposure HDR for more dynamic range
	//   Best for: High-contrast scenes, sunset/sunrise shots
	// - 7: 7-exposure HDR for maximum detail in shadows and highlights
	//   Best for: Extreme lighting (bright windows + dark interiors)
	//
	// Note: More brackets = better HDR but longer processing time
	// Default: 3 (only applies when bracket_grouping="auto")
	BracketsPerImage int `json:"brackets_per_image,omitempty"`

	// Optional metadata to store with the processing request for your own tracking
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
