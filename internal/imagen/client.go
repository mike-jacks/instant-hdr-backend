package imagen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Data struct {
		FilesList []struct {
			FileName   string `json:"file_name"`
			UploadLink string `json:"upload_link"`
		} `json:"files_list"`
	} `json:"data"`
}

type EditRequest struct {
	ProfileKey  string                 `json:"profile_key,omitempty"`
	HDRMerge    bool                   `json:"hdr_merge"`
	JPEGExport  bool                   `json:"jpeg_export"`
	AITools     []string               `json:"ai_tools,omitempty"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type EditResponse struct {
	Data struct {
		EditID string `json:"edit_id"`
	} `json:"data"`
}

type EditStatusResponse struct {
	Data struct {
		Status   string `json:"status"`
		Progress int    `json:"progress"`
	} `json:"data"`
}

type ExportResponse struct {
	Data struct {
		DownloadURL string `json:"download_url"`
	} `json:"data"`
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
	req, err := http.NewRequest("POST", c.baseURL+"/projects/", nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result CreateProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
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

	req, err := http.NewRequest("POST", c.baseURL+"/projects/"+projectUUID+"/get_temporary_upload_links", bytes.NewBuffer(jsonData))
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

	uploadLinks := make([]string, len(result.Data.FilesList))
	for i, file := range result.Data.FilesList {
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

func (c *Client) Edit(projectUUID string, editReq EditRequest) (string, error) {
	// Always set jpeg_export to true
	editReq.JPEGExport = true

	jsonData, err := json.Marshal(editReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/projects/"+projectUUID+"/edit", bytes.NewBuffer(jsonData))
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to edit project: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result EditResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.EditID, nil
}

func (c *Client) GetEditStatus(projectUUID, editID string) (*EditStatusResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/projects/"+projectUUID+"/edit/"+editID+"/status", nil)
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

func (c *Client) Export(projectUUID string) (string, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/projects/"+projectUUID+"/export", nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to export project: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result ExportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.DownloadURL, nil
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

func (c *Client) DeleteProject(projectUUID string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/projects/"+projectUUID, nil)
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
