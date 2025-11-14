package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type UploadHandler struct {
	imagenClient   *imagen.Client
	dbClient       *supabase.DatabaseClient
	realtimeClient *supabase.RealtimeClient
}

func NewUploadHandler(imagenClient *imagen.Client, dbClient *supabase.DatabaseClient, realtimeClient *supabase.RealtimeClient) *UploadHandler {
	return &UploadHandler{
		imagenClient:   imagenClient,
		dbClient:       dbClient,
		realtimeClient: realtimeClient,
	}
}

// Upload godoc
// @Summary     Upload images to project
// @Description Uploads multiple bracketed images to an Imagen AI project. All images in a single upload are expected to be bracketed images of the same shot.
// @Tags        upload
// @Accept      multipart/form-data
// @Produce     json
// @Security    Bearer
// @Param       project_id path string true "Project ID (UUID)"
// @Param       images formData file true "Bracketed images (multiple files allowed)"
// @Success     200 {object} models.UploadResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /projects/{project_id}/upload [post]
func (h *UploadHandler) Upload(c *gin.Context) {
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

	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid project id"})
		return
	}

	// Verify project belongs to user
	project, err := h.dbClient.GetProject(projectID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "project not found",
			Message: err.Error(),
		})
		return
	}

	// Set max memory for multipart form (32MB)
	err = c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse multipart form",
			Message: err.Error(),
		})
		return
	}

	// Parse multipart form
	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to parse multipart form",
			Message: "multipart form is nil",
		})
		return
	}

	// Try multiple common field names
	var files []*multipart.FileHeader
	fieldNames := []string{"images", "image", "files", "file", "photos", "photo"}
	for _, fieldName := range fieldNames {
		if f := form.File[fieldName]; len(f) > 0 {
			files = f
			break
		}
	}

	if len(files) == 0 {
		// Get all available field names for debugging
		availableFields := make([]string, 0, len(form.File))
		for fieldName := range form.File {
			availableFields = append(availableFields, fieldName)
		}
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "no files uploaded",
			Message: fmt.Sprintf("please provide files with one of these field names: %v. Available fields in request: %v", fieldNames, availableFields),
		})
		return
	}

	// Publish upload_started event
	h.realtimeClient.PublishProjectEvent(projectID, "upload_started",
		supabase.UploadStartedPayload(projectID, len(files)))

	// Update status
	h.dbClient.UpdateProjectStatus(projectID, "uploading", 0)

	// Get filenames
	filenames := make([]string, len(files))
	for i, file := range files {
		filenames[i] = file.Filename
	}

	// Get upload links from Imagen
	var uploadLinks []string
	err = h.imagenClient.RetryWithBackoff(func() error {
		var err error
		uploadLinks, err = h.imagenClient.GetUploadLinks(project.ImagenProjectUUID, filenames)
		return err
	}, 3)
	if err != nil {
		h.dbClient.UpdateProjectError(projectID, fmt.Sprintf("failed to get upload links: %v", err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get upload links",
			Message: err.Error(),
		})
		return
	}

	// Validate upload links match file count
	if len(uploadLinks) != len(files) {
		h.dbClient.UpdateProjectError(projectID, fmt.Sprintf("upload links count mismatch: got %d links for %d files", len(uploadLinks), len(files)))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload links mismatch",
			Message: fmt.Sprintf("received %d upload links but have %d files", len(uploadLinks), len(files)),
		})
		return
	}

	// Upload files to Imagen
	uploadedFiles := make([]models.FileInfo, 0)
	uploadErrors := make([]string, 0)
	for i, file := range files {
		// Open file
		src, err := file.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to open: %v", file.Filename, err))
			continue
		}

		// Read file data
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to read: %v", file.Filename, err))
			continue
		}

		// Upload to Imagen with retry
		err = h.imagenClient.RetryWithBackoff(func() error {
			return h.imagenClient.UploadFile(uploadLinks[i], data)
		}, 3)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to upload: %v", file.Filename, err))
			continue
		}

		uploadedFiles = append(uploadedFiles, models.FileInfo{
			Filename: file.Filename,
			Size:     file.Size,
		})
	}

	if len(uploadedFiles) == 0 {
		errorMsg := "failed to upload any files"
		if len(uploadErrors) > 0 {
			errorMsg += ": " + fmt.Sprintf("%v", uploadErrors)
		}
		h.dbClient.UpdateProjectError(projectID, errorMsg)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to upload files",
			Message: errorMsg,
		})
		return
	}

	// Update status
	h.dbClient.UpdateProjectStatus(projectID, "uploaded", 0)

	// Publish upload_completed event
	h.realtimeClient.PublishProjectEvent(projectID, "upload_completed",
		supabase.UploadCompletedPayload(projectID, len(uploadedFiles)))

	c.JSON(http.StatusOK, models.UploadResponse{
		ProjectID: projectID.String(),
		Files:     uploadedFiles,
		Status:    "uploaded",
	})
}
