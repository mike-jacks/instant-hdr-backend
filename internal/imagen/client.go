package imagen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type CreateProjectResponse struct {
	Data struct {
		ProjectUUID string `json:"project_uuid"`
	} `json:"data"`
}

type UploadLinkRequest struct {
	FilesList []struct {
		FileName string `json:"file_name"`
	} `json:"files_list"`
}

type UploadLinkResponse struct {
	FilesList []struct {
		FileName   string `json:"file_name"`
		UploadLink string `json:"upload_link"`
	} `json:"files_list"`
}

type EditRequest struct {
	ProfileKey              int                    `json:"profile_key"`
	Crop                    bool                   `json:"crop,omitempty"`
	PortraitCrop            bool                   `json:"portrait_crop,omitempty"`
	HeadshotCrop            bool                   `json:"headshot_crop,omitempty"`
	CropAspectRatio         string                 `json:"crop_aspect_ratio,omitempty"` // "2X3", "4X5", "5X7"
	HDRMerge                bool                   `json:"hdr_merge,omitempty"`
	Straighten              bool                   `json:"straighten,omitempty"`
	SubjectMask             bool                   `json:"subject_mask,omitempty"`
	PhotographyType         string                 `json:"photography_type,omitempty"` // "NO_TYPE", "REAL_ESTATE", etc.
	CallbackURL             string                 `json:"callback_url,omitempty"`
	SmoothSkin              bool                   `json:"smooth_skin,omitempty"`
	PerspectiveCorrection    bool                   `json:"perspective_correction,omitempty"`
	WindowPull              bool                   `json:"window_pull,omitempty"`
	SkyReplacement          bool                   `json:"sky_replacement,omitempty"`
	SkyReplacementTemplateID int                    `json:"sky_replacement_template_id,omitempty"`
	HDROutputCompression    string                 `json:"hdr_output_compression,omitempty"` // "LOSSY", "LOSSLESS"
}

// EditResponse is empty according to OpenAPI spec - no response body

type EditStatusResponse struct {
	Status string `json:"status"` // "Pending", "In Progress", "Failed", "Completed"
}

type ExportResponse struct {
	ProjectUUID string `json:"project_uuid"`
	Message     string `json:"message"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateProject() (string, error) {
	// According to OpenAPI spec: POST /v1/projects/ or /v1/projects
	// Try with trailing slash first (as shown in OpenAPI spec)
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/"
	
	// Send empty JSON body (request body is optional but some APIs expect it)
	jsonData := []byte("{}")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result CreateProjectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	if result.Data.ProjectUUID == "" {
		return "", fmt.Errorf("project_uuid is empty in response, body: %s", string(body))
	}

	return result.Data.ProjectUUID, nil
}

func (c *Client) GetUploadLinks(projectUUID string, filenames []string) ([]string, error) {
	filesList := make([]struct {
		FileName string `json:"file_name"`
	}, len(filenames))
	for i, filename := range filenames {
		filesList[i].FileName = filename
	}

	requestBody := UploadLinkRequest{
		FilesList: filesList,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/get_temporary_upload_links"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get upload links: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result UploadLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	uploadLinks := make([]string, len(result.FilesList))
	for i, file := range result.FilesList {
		uploadLinks[i] = file.UploadLink
	}

	return uploadLinks, nil
}

func (c *Client) UploadFile(uploadLink string, data []byte) error {
	req, err := http.NewRequest("PUT", uploadLink, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "")

	resp, err := c.httpClient.Do(req)
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

func (c *Client) Edit(projectUUID string, editReq EditRequest) error {
	jsonData, err := json.Marshal(editReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/edit"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to edit project: status %d, body: %s", resp.StatusCode, string(body))
	}

	// OpenAPI spec shows empty response body for edit endpoint
	return nil
}

func (c *Client) GetEditStatus(projectUUID string) (*EditStatusResponse, error) {
	// According to OpenAPI spec: GET /v1/projects/{project_uuid}/edit/status
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/edit/status"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get edit status: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result EditStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Export(projectUUID string) error {
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/export"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to export project: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result ExportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Export just submits the request, doesn't return download URL
	// Use GetExportDownloadLinks to get the download links after checking status
	return nil
}

func (c *Client) DownloadFile(downloadURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to download file: status %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// GetEditDownloadLinks returns temporary download links for edited files
func (c *Client) GetEditDownloadLinks(projectUUID string) ([]struct {
	FileName     string `json:"file_name"`
	DownloadLink string `json:"download_link"`
}, error) {
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/edit/get_temporary_download_links"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get edit download links: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		FilesList []struct {
			FileName     string `json:"file_name"`
			DownloadLink string `json:"download_link"`
		} `json:"files_list"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.FilesList, nil
}

// GetExportStatus returns the export status for a project
func (c *Client) GetExportStatus(projectUUID string) (*struct {
	ProjectUUID string `json:"project_uuid"`
	Status      string `json:"status"` // "Pending", "In Progress", "Failed", "Completed"
}, error) {
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/export/status"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get export status: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ProjectUUID string `json:"project_uuid"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetExportDownloadLinks returns temporary download links for exported files
func (c *Client) GetExportDownloadLinks(projectUUID string) ([]struct {
	FileName     string `json:"file_name"`
	DownloadLink string `json:"download_link"`
}, error) {
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID + "/export/get_temporary_download_links"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get export download links: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		FilesList []struct {
			FileName     string `json:"file_name"`
			DownloadLink string `json:"download_link"`
		} `json:"files_list"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.FilesList, nil
}

func (c *Client) DeleteProject(projectUUID string) error {
	url := strings.TrimSuffix(c.baseURL, "/") + "/projects/" + projectUUID
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete project: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
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
