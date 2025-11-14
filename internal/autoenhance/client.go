package autoenhance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AutoEnhanceTime is a custom time type that handles timestamps without timezone
type AutoEnhanceTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for AutoEnhanceTime
func (t *AutoEnhanceTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	str := strings.Trim(string(data), `"`)
	
	// Try parsing with various formats
	formats := []string{
		"2006-01-02T15:04:05.999999",           // Without timezone (AutoEnhance format)
		"2006-01-02T15:04:05.999999Z",         // With Z
		"2006-01-02T15:04:05.999999Z07:00",    // With timezone
		"2006-01-02T15:04:05Z07:00",           // RFC3339
		"2006-01-02T15:04:05",                 // Without microseconds
		time.RFC3339,                           // Standard RFC3339
		time.RFC3339Nano,                       // RFC3339 with nanoseconds
	}
	
	var err error
	for _, format := range formats {
		t.Time, err = time.Parse(format, str)
		if err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("failed to parse time: %s", str)
}

// MarshalJSON implements json.Marshaler for AutoEnhanceTime
func (t AutoEnhanceTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339))
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// OrderIn represents the request body for creating an order
type OrderIn struct {
	Name    string `json:"name,omitempty"`
	OrderID string `json:"order_id,omitempty"`
}

// OrderOut represents the response from order operations
type OrderOut struct {
	OrderID       string         `json:"order_id"`
	Name          string         `json:"name"`
	Status        string         `json:"status"`
	IsProcessing  bool           `json:"is_processing"`
	IsMerging     bool           `json:"is_merging"`
	IsDeleted     bool           `json:"is_deleted"`
	TotalImages   float64        `json:"total_images"`
	CreatedAt     AutoEnhanceTime `json:"created_at"`
	LastUpdatedAt AutoEnhanceTime `json:"last_updated_at"`
	Images        []ImageOut     `json:"images"`
}

// OrdersOut represents the response from listing orders
type OrdersOut struct {
	Orders    []OrderOut `json:"orders"`
	Pagination struct {
		NextOffset string `json:"next_offset"`
		PerPage    int    `json:"per_page"`
	} `json:"pagination"`
}

// BracketIn represents the request body for creating a bracket
type BracketIn struct {
	Name     string                 `json:"name"`
	OrderID  string                 `json:"order_id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BracketCreatedOut represents the response from creating a bracket
type BracketCreatedOut struct {
	BracketID  string                 `json:"bracket_id"`
	ImageID    string                 `json:"image_id,omitempty"`
	OrderID    string                 `json:"order_id,omitempty"`
	Name       string                 `json:"name"`
	UploadURL  string                 `json:"upload_url,omitempty"`
	IsUploaded bool                   `json:"is_uploaded"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BracketOut represents a bracket in responses
type BracketOut struct {
	BracketID  string                 `json:"bracket_id"`
	ImageID    string                 `json:"image_id,omitempty"`
	OrderID    string                 `json:"order_id,omitempty"`
	Name       string                 `json:"name"`
	IsUploaded bool                   `json:"is_uploaded"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// OrderBracketsOut represents the response from getting order brackets
type OrderBracketsOut struct {
	Brackets []BracketOut `json:"brackets"`
}

// OrderImageIn represents an image in the process request
type OrderImageIn struct {
	BracketIDs []string `json:"bracket_ids"`
}

// OrderHDRProcessIn represents the request body for processing an order
type OrderHDRProcessIn struct {
	EnhanceType            string        `json:"enhance_type,omitempty"` // "property", "property_usa", "warm", "neutral", "modern"
	SkyReplacement         *bool         `json:"sky_replacement,omitempty"`
	VerticalCorrection     *bool         `json:"vertical_correction,omitempty"`
	LensCorrection         *bool         `json:"lens_correction,omitempty"`
	WindowPullType         *string        `json:"window_pull_type,omitempty"` // "NONE", "ONLY_WINDOWS", "WINDOWS_WITH_SKIES"
	Upscale                *bool         `json:"upscale,omitempty"`
	Privacy                *bool         `json:"privacy,omitempty"`
	CloudType              *string       `json:"cloud_type,omitempty"` // "CLEAR", "LOW_CLOUD", "HIGH_CLOUD"
	AIVersion              string        `json:"ai_version,omitempty"`
	Enhance                *bool         `json:"enhance,omitempty"`
	NumberOfBracketsPerImage *int         `json:"number_of_brackets_per_image,omitempty"`
	Images                 []OrderImageIn `json:"images,omitempty"`
}

// OrderHDRProcessOut represents the response from processing an order
type OrderHDRProcessOut struct {
	OrderID       string     `json:"order_id"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	IsProcessing  bool       `json:"is_processing"`
	IsMerging     bool       `json:"is_merging"`
	IsDeleted     bool       `json:"is_deleted"`
	TotalImages   float64    `json:"total_images"`
	CreatedAt     AutoEnhanceTime `json:"created_at"`
	LastUpdatedAt AutoEnhanceTime `json:"last_updated_at"`
	Images        []ImageOut `json:"images"`
}

// ImageOut represents an image in responses
type ImageOut struct {
	ImageID          string                 `json:"image_id"`
	ImageName         string                 `json:"image_name"`
	OrderID           string                 `json:"order_id,omitempty"`
	Status            string                 `json:"status,omitempty"`
	StatusReason      string                 `json:"status_reason,omitempty"`
	EnhanceType       string                 `json:"enhance_type,omitempty"`
	Enhance           bool                   `json:"enhance,omitempty"`
	SkyReplacement    bool                   `json:"sky_replacement,omitempty"`
	VerticalCorrection bool                  `json:"vertical_correction,omitempty"`
	LensCorrection    bool                   `json:"lens_correction,omitempty"`
	WindowPullType    *string                `json:"window_pull_type,omitempty"`
	Upscale           bool                   `json:"upscale,omitempty"`
	Privacy           *bool                  `json:"privacy,omitempty"`
	CloudType         *string                `json:"cloud_type,omitempty"`
	AIVersion         string                 `json:"ai_version,omitempty"`
	Downloaded        bool                   `json:"downloaded,omitempty"`
	DateAdded         int64                  `json:"date_added,omitempty"`
	Scene             string                 `json:"scene,omitempty"`
	Rating            *int                   `json:"rating,omitempty"`
	PresetID          string                 `json:"preset_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	UserID            string                 `json:"user_id,omitempty"`
}

// DownloadOptions represents options for downloading images
type DownloadOptions struct {
	Format   string  // "png", "jpeg", "webp"
	Preview  *bool
	Watermark *bool
	Finetune *bool
	MaxWidth *int
	Scale    *float64
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateOrder creates a new order in AutoEnhance
func (c *Client) CreateOrder(orderID, name string) (*OrderOut, error) {
	reqBody := OrderIn{
		OrderID: orderID,
		Name:    name,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v3/orders/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create order: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrderOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// GetOrder retrieves an order by ID
func (c *Client) GetOrder(orderID string) (*OrderOut, error) {
	url := c.baseURL + "/v3/orders/" + orderID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get order: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrderOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// UpdateOrder updates an order
func (c *Client) UpdateOrder(orderID string, orderIn OrderIn) (*OrderOut, error) {
	jsonData, err := json.Marshal(orderIn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v3/orders/" + orderID
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update order: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrderOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// DeleteOrder deletes an order
func (c *Client) DeleteOrder(orderID string) error {
	url := c.baseURL + "/v3/orders/" + orderID
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete order: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListOrders lists orders with pagination
func (c *Client) ListOrders(offset string, perPage int) (*OrdersOut, error) {
	endpointURL := c.baseURL + "/v3/orders/"
	if offset != "" || perPage > 0 {
		params := url.Values{}
		if offset != "" {
			params.Add("offset", offset)
		}
		if perPage > 0 {
			params.Add("per_page", fmt.Sprintf("%d", perPage))
		}
		endpointURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list orders: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrdersOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// CreateBracket creates a new bracket in an order
func (c *Client) CreateBracket(bracketIn BracketIn) (*BracketCreatedOut, error) {
	jsonData, err := json.Marshal(bracketIn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v3/brackets/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create bracket: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result BracketCreatedOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// GetBracket retrieves a bracket by ID
func (c *Client) GetBracket(bracketID string) (*BracketOut, error) {
	url := c.baseURL + "/v3/brackets/" + bracketID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bracket: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result BracketOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// GetOrderBrackets retrieves all brackets for an order
func (c *Client) GetOrderBrackets(orderID string) (*OrderBracketsOut, error) {
	url := c.baseURL + "/v3/orders/" + orderID + "/brackets"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get order brackets: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrderBracketsOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// DeleteBracket deletes a bracket
func (c *Client) DeleteBracket(bracketID string) error {
	url := c.baseURL + "/v3/brackets/" + bracketID
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete bracket: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UploadFile uploads a file to the provided upload URL
// According to AutoEnhance.ai docs: https://docs.autoenhance.ai/
// The Content-Type header should be set to "application/octet-stream"
func (c *Client) UploadFile(uploadURL string, data []byte, mimeType string) error {
	// Parse the URL to extract headers that are part of the signature
	parsedURL, err := url.Parse(uploadURL)
	if err != nil {
		return fmt.Errorf("failed to parse upload URL: %w", err)
	}

	// Create request with body
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// According to AutoEnhance.ai documentation, set Content-Type to application/octet-stream
	// https://docs.autoenhance.ai/ - "Set the Content-Type to application/octet-stream during the upload"
	req.Header.Set("Content-Type", "application/octet-stream")

	// Extract headers from query parameters that are part of the signature
	// S3 pre-signed URLs include x-amz-* headers in query params, but they must
	// also be present as request headers for the signature to match
	query := parsedURL.Query()
	
	// Extract x-amz-meta-bracket_id if present (URL decode it)
	if bracketID := query.Get("x-amz-meta-bracket_id"); bracketID != "" {
		decoded, err := url.QueryUnescape(bracketID)
		if err == nil {
			req.Header.Set("x-amz-meta-bracket_id", decoded)
		} else {
			req.Header.Set("x-amz-meta-bracket_id", bracketID)
		}
	}
	
	// Extract x-amz-security-token if present (URL decode it)
	if securityToken := query.Get("x-amz-security-token"); securityToken != "" {
		decoded, err := url.QueryUnescape(securityToken)
		if err == nil {
			req.Header.Set("x-amz-security-token", decoded)
		} else {
			req.Header.Set("x-amz-security-token", securityToken)
		}
	}

	// Remove any headers that Go might add automatically
	req.Header.Del("User-Agent")
	req.Header.Del("Accept-Encoding")

	// Use a custom client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload file: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ProcessOrder processes an order with HDR merging
func (c *Client) ProcessOrder(orderID string, processIn OrderHDRProcessIn) (*OrderHDRProcessOut, error) {
	jsonData, err := json.Marshal(processIn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v3/orders/" + orderID + "/process"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to process order: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result OrderHDRProcessOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// GetImage retrieves an image by ID
func (c *Client) GetImage(imageID string) (*ImageOut, error) {
	url := c.baseURL + "/v3/images/" + imageID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get image: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result ImageOut
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// DownloadEnhanced downloads the enhanced version of an image
func (c *Client) DownloadEnhanced(imageID string, options DownloadOptions) ([]byte, error) {
	endpointURL := c.baseURL + "/v3/images/" + imageID + "/enhanced"
	
	params := url.Values{}
	if options.Format != "" {
		params.Add("format", options.Format)
	}
	if options.Preview != nil {
		params.Add("preview", fmt.Sprintf("%t", *options.Preview))
	}
	if options.Watermark != nil {
		params.Add("watermark", fmt.Sprintf("%t", *options.Watermark))
	}
	if options.Finetune != nil {
		params.Add("finetune", fmt.Sprintf("%t", *options.Finetune))
	}
	if options.MaxWidth != nil {
		params.Add("max_width", fmt.Sprintf("%d", *options.MaxWidth))
	}
	if options.Scale != nil {
		params.Add("scale", fmt.Sprintf("%f", *options.Scale))
	}
	
	if len(params) > 0 {
		endpointURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download enhanced image: status %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// DownloadOriginal downloads the original version of an image
func (c *Client) DownloadOriginal(imageID string, options DownloadOptions) ([]byte, error) {
	endpointURL := c.baseURL + "/v3/images/" + imageID + "/original"
	
	params := url.Values{}
	if options.Format != "" {
		params.Add("format", options.Format)
	}
	if options.Preview != nil {
		params.Add("preview", fmt.Sprintf("%t", *options.Preview))
	}
	if options.Watermark != nil {
		params.Add("watermark", fmt.Sprintf("%t", *options.Watermark))
	}
	if options.Finetune != nil {
		params.Add("finetune", fmt.Sprintf("%t", *options.Finetune))
	}
	if options.MaxWidth != nil {
		params.Add("max_width", fmt.Sprintf("%d", *options.MaxWidth))
	}
	if options.Scale != nil {
		params.Add("scale", fmt.Sprintf("%f", *options.Scale))
	}
	
	if len(params) > 0 {
		endpointURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download original image: status %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func (c *Client) RetryWithBackoff(fn func() error, maxRetries int) error {
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if i < len(backoffs) {
			time.Sleep(backoffs[i])
		}
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

