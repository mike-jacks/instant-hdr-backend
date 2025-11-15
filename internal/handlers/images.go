package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type ImagesHandler struct {
	autoenhanceClient *autoenhance.Client
	dbClient          *supabase.DatabaseClient
	storageClient     *supabase.StorageClient
}

func NewImagesHandler(autoenhanceClient *autoenhance.Client, dbClient *supabase.DatabaseClient, storageClient *supabase.StorageClient) *ImagesHandler {
	return &ImagesHandler{
		autoenhanceClient: autoenhanceClient,
		dbClient:          dbClient,
		storageClient:     storageClient,
	}
}

// ListImages godoc
// @Summary     List processed images
// @Description Returns a list of all processed images for an order with their download status
// @Tags        images
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} models.ImagesResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/images [get]
func (h *ImagesHandler) ListImages(c *gin.Context) {
	if h.dbClient == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database not available"})
		return
	}

	userIDStr, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid user id"})
		return
	}

	orderIDStr := c.Param("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid order id"})
		return
	}

	// Verify order belongs to user
	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Get images from AutoEnhance
	autoenhanceOrder, err := h.autoenhanceClient.GetOrder(order.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get order from AutoEnhance",
			Message: err.Error(),
		})
		return
	}

	// Get processed files from our database to check what's already downloaded
	dbFiles, _ := h.dbClient.GetOrderFiles(orderID, userID)
	
	// Create a map of downloaded files for quick lookup
	downloadedFiles := make(map[string]models.FileResponse)
	for _, file := range dbFiles {
		// Key format: imageID_quality (e.g., "img123_preview" or "img123_high")
		key := extractImageIDFromFilename(file.Filename)
		downloadedFiles[key] = models.FileResponse{
			ID:         file.ID.String(),
			Filename:   file.Filename,
			StorageURL: file.StorageURL,
			FileSize:   file.FileSize.Int64,
			MimeType:   file.MimeType,
			IsFinal:    file.IsFinal,
			CreatedAt:  file.CreatedAt,
		}
	}

	// Build response with images from AutoEnhance
	imageResponses := make([]models.ImageResponse, 0, len(autoenhanceOrder.Images))
	for _, img := range autoenhanceOrder.Images {
		imageResp := models.ImageResponse{
			ImageID:     img.ImageID,
			ImageName:   img.ImageName,
			Status:      img.Status,
			EnhanceType: img.EnhanceType,
			Downloaded:  img.Downloaded,
		}

		// Check if preview is downloaded
		previewKey := fmt.Sprintf("%s_preview", img.ImageID)
		if previewFile, exists := downloadedFiles[previewKey]; exists {
			imageResp.PreviewDownloaded = true
			imageResp.PreviewURL = previewFile.StorageURL
		}

		// Check if high-res is downloaded
		highResKey := fmt.Sprintf("%s_high", img.ImageID)
		if highResFile, exists := downloadedFiles[highResKey]; exists {
			imageResp.HighResDownloaded = true
			imageResp.HighResURL = highResFile.StorageURL
		}

		// Add processing settings
		settings := make(map[string]interface{})
		if img.EnhanceType != "" {
			settings["enhance_type"] = img.EnhanceType
		}
		if img.SkyReplacement {
			settings["sky_replacement"] = img.SkyReplacement
		}
		if img.VerticalCorrection {
			settings["vertical_correction"] = img.VerticalCorrection
		}
		if img.LensCorrection {
			settings["lens_correction"] = img.LensCorrection
		}
		if img.WindowPullType != nil {
			settings["window_pull_type"] = *img.WindowPullType
		}
		if len(settings) > 0 {
			imageResp.ProcessingSettings = settings
		}

		imageResponses = append(imageResponses, imageResp)
	}

	c.JSON(http.StatusOK, models.ImagesResponse{Images: imageResponses})
}

// DownloadImage godoc
// @Summary     Download processed image to Supabase Storage
// @Description Downloads a processed image from AutoEnhance and stores it in Supabase Storage.
// @Description
// @Description Quality Options:
// @Description - "thumbnail": 400px width (~50-100KB) - List view
// @Description - "preview": 800px width (~150-250KB) - Gallery view (DEFAULT)
// @Description - "medium": 1920px width (~500KB-1MB) - Full screen
// @Description - "high": Full resolution (~2-5MB) - Client delivery
// @Description - "custom": Specify max_width or scale
// @Description
// @Description Format Options: "jpeg" (default), "png", "webp"
// @Description
// @Description Watermark (defaults to true = FREE):
// @Description - true: FREE download with watermark
// @Description - false: COSTS 1 CREDIT (unwatermarked)
// @Tags        images
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Param       image_id path string true "Image ID from AutoEnhance"
// @Param       request body models.DownloadImageRequest true "Download options"
// @Success     200 {object} models.DownloadImageResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/images/{image_id}/download [post]
func (h *ImagesHandler) DownloadImage(c *gin.Context) {
	if h.dbClient == nil || h.storageClient == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "storage not available"})
		return
	}

	userIDStr, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid user id"})
		return
	}

	orderIDStr := c.Param("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid order id"})
		return
	}

	imageID := c.Param("image_id")
	if imageID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "image_id is required"})
		return
	}

	var req models.DownloadImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Default quality if not specified
	if req.Quality == "" {
		req.Quality = "preview"
	}

	// Validate quality parameter
	validQualities := map[string]bool{
		"thumbnail": true,
		"preview":   true,
		"medium":    true,
		"high":      true,
		"custom":    true,
	}
	if !validQualities[req.Quality] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "quality must be one of: thumbnail, preview, medium, high, custom",
		})
		return
	}

	// Validate custom options
	if req.Quality == "custom" && req.MaxWidth == nil && req.Scale == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "custom quality requires either max_width or scale",
		})
		return
	}

	// Verify order belongs to user
	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Get image info from AutoEnhance to verify it exists
	_, err = h.autoenhanceClient.GetImage(imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "image not found",
			Message: err.Error(),
		})
		return
	}

	// Default watermark to true (FREE) if not specified
	watermark := true
	if req.Watermark != nil {
		watermark = *req.Watermark
	}
	
	// Set download options based on quality
	options := autoenhance.DownloadOptions{
		Format:    "jpeg", // Default format
		Watermark: &watermark, // Defaults to true (FREE), but can be overridden
	}

	// Allow custom format if specified
	if req.Format != "" {
		validFormats := map[string]bool{"jpeg": true, "png": true, "webp": true}
		if validFormats[req.Format] {
			options.Format = req.Format
		}
	}

	var resolution string

	// Map quality presets to settings
	switch req.Quality {
	case "thumbnail":
		maxWidth := 400
		options.MaxWidth = &maxWidth
		resolution = "400px"

	case "preview":
		maxWidth := 800
		options.MaxWidth = &maxWidth
		resolution = "800px"

	case "medium":
		maxWidth := 1920
		options.MaxWidth = &maxWidth
		resolution = "1920px"

	case "high":
		// Full resolution - no maxWidth limit
		resolution = "full"

	case "custom":
		if req.MaxWidth != nil {
			options.MaxWidth = req.MaxWidth
			resolution = fmt.Sprintf("%dpx", *req.MaxWidth)
		}
		if req.Scale != nil {
			options.Scale = req.Scale
			resolution = fmt.Sprintf("%.0f%%", *req.Scale*100)
		}
	}

	// Download from AutoEnhance
	var imageData []byte
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		data, err := h.autoenhanceClient.DownloadEnhanced(imageID, options)
		if err != nil {
			return err
		}
		imageData = data
		return nil
	}, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to download image from AutoEnhance",
			Message: err.Error(),
		})
		return
	}

	// Generate filename: {image_id}_{quality}.jpg
	filename := fmt.Sprintf("%s_%s.jpg", imageID, req.Quality)

	// Upload to Supabase Storage
	_, publicURL, err := h.storageClient.UploadFile(userID, orderID, filename, imageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to upload to storage",
			Message: err.Error(),
		})
		return
	}

	// Store file record in database
	orderFile := &models.OrderFile{
		ID:         uuid.New(),
		OrderID:    order.ID,
		Filename:   filename,
		StorageURL: publicURL,
		MimeType:   "image/jpeg",
		IsFinal:    true,
	}

	if err := h.dbClient.CreateOrderFile(orderFile); err != nil {
		// Log error but don't fail - file is already in storage
		// The file will still be accessible via the URL
	}

	// Determine if credit was used
	creditUsed := !watermark

	// Build appropriate message
	var message string
	if watermark {
		message = fmt.Sprintf("Image downloaded successfully (FREE with watermark) - Quality: %s, Resolution: %s", req.Quality, resolution)
	} else {
		message = fmt.Sprintf("Image downloaded successfully (1 CREDIT USED - unwatermarked) - Quality: %s, Resolution: %s", req.Quality, resolution)
	}

	c.JSON(http.StatusOK, models.DownloadImageResponse{
		ImageID:    imageID,
		Quality:    req.Quality,
		URL:        publicURL,
		FileSize:   int64(len(imageData)),
		Watermark:  watermark,
		Resolution: resolution,
		Format:     options.Format,
		CreditUsed: creditUsed,
		Message:    message,
	})
}

// Helper function to extract image ID from filename
// Filename format: {image_id}_{quality}.jpg
func extractImageIDFromFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".jpg")
	name = strings.TrimSuffix(name, ".jpeg")
	name = strings.TrimSuffix(name, ".png")
	
	return name
}

