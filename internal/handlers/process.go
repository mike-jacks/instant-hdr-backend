package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type ProcessHandler struct {
	imagenClient   *imagen.Client
	dbClient       *supabase.DatabaseClient
	realtimeClient *supabase.RealtimeClient
	webhookURL     string
}

func NewProcessHandler(imagenClient *imagen.Client, dbClient *supabase.DatabaseClient, realtimeClient *supabase.RealtimeClient, webhookURL string) *ProcessHandler {
	return &ProcessHandler{
		imagenClient:   imagenClient,
		dbClient:       dbClient,
		realtimeClient: realtimeClient,
		webhookURL:     webhookURL,
	}
}

// Process godoc
// @Summary     Process images with HDR merge
// @Description Initiates HDR processing and merging of uploaded images using Imagen AI. Automatically enables JPEG export.
// @Tags        process
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       project_id path string true "Project ID (UUID)"
// @Param       request body models.ProcessRequest true "Processing options"
// @Success     200 {object} models.ProcessResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /projects/{project_id}/process [post]
func (h *ProcessHandler) Process(c *gin.Context) {
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

	var req models.ProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Build edit request
	editReq := imagen.EditRequest{
		ProfileKey:  req.ProfileKey,
		HDRMerge:    req.HDRMerge,
		JPEGExport:  true, // Always true
		AITools:     req.AITools,
		CallbackURL: h.webhookURL,
		Metadata:    req.Metadata,
	}

	// Initiate processing with retry
	var editID string
	err = h.imagenClient.RetryWithBackoff(func() error {
		var err error
		editID, err = h.imagenClient.Edit(project.ImagenProjectUUID, editReq)
		return err
	}, 3)
	if err != nil {
		h.dbClient.UpdateProjectError(projectID, err.Error())
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to initiate processing",
			Message: err.Error(),
		})
		return
	}

	// Update project with edit_id
	h.dbClient.UpdateProjectEditID(projectID, editID)
	h.dbClient.UpdateProjectStatus(projectID, "processing", 0)

	// Publish processing_started event
	h.realtimeClient.PublishProjectEvent(projectID, "processing_started",
		supabase.ProcessingStartedPayload(projectID, editID))

	c.JSON(http.StatusOK, models.ProcessResponse{
		ProjectID: projectID.String(),
		Status:    "processing",
		EditID:    editID,
	})
}
