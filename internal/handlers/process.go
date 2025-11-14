package handlers

import (
	"fmt"
	"net/http"
	"strings"

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
// @Description Initiates HDR processing and merging of uploaded images using Imagen AI. If profile_key is not provided, the system will automatically search for and use a profile named "NATURAL HOME - JPEG" (or similar). HDR merge is enabled by default for real estate photography.
// @Tags        process
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       project_id path string true "Project ID (UUID)"
// @Param       request body models.ProcessRequest false "Processing options. profile_key is optional - if not provided, will auto-select 'NATURAL HOME - JPEG' profile. profile_key should be an integer (can be sent as string, will be converted). Get available profiles from /profiles endpoint."
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

	// Build edit request according to OpenAPI spec
	// Get or determine profile_key
	var profileKey int
	if req.ProfileKey != "" {
		// User provided profile_key, use it
		_, err := fmt.Sscanf(req.ProfileKey, "%d", &profileKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid profile_key",
				Message: "profile_key must be a valid integer",
			})
			return
		}
	} else {
		// No profile_key provided, try to find "natural home" profile
		profiles, err := h.imagenClient.GetUserProfiles()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "failed to get profiles",
				Message: "could not retrieve profiles to find default. Please provide profile_key in request.",
			})
			return
		}

		// Search for "NATURAL HOME - JPEG" profile (case-insensitive)
		// Also matches variations like "Natural Home - JPEG", "natural home - jpeg", etc.
		found := false
		for _, profile := range profiles {
			profileName := strings.ToLower(profile.ProfileName)
			// Match "natural home" (with or without "jpeg" suffix)
			if strings.Contains(profileName, "natural") && strings.Contains(profileName, "home") {
				profileKey = profile.ProfileKey
				found = true
				break
			}
		}

		if !found {
			// If "natural home" not found, use first available profile
			if len(profiles) > 0 {
				profileKey = profiles[0].ProfileKey
			} else {
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "no profile available",
					Message: "no profiles found. Please create a profile in Imagen AI or provide profile_key in request.",
				})
				return
			}
		}
	}

	editReq := imagen.EditRequest{
		ProfileKey:      profileKey,
		HDRMerge:        req.HDRMerge,
		CallbackURL:     h.webhookURL,
		PhotographyType: "REAL_ESTATE", // Default for real estate shoots
	}

	// Initiate processing with retry
	err = h.imagenClient.RetryWithBackoff(func() error {
		return h.imagenClient.Edit(project.ImagenProjectUUID, editReq)
	}, 3)
	if err != nil {
		h.dbClient.UpdateProjectError(projectID, err.Error())
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to initiate processing",
			Message: err.Error(),
		})
		return
	}

	// Update project status (no editID returned from API)
	h.dbClient.UpdateProjectStatus(projectID, "processing", 0)

	// Publish processing_started event
	h.realtimeClient.PublishProjectEvent(projectID, "processing_started",
		supabase.ProcessingStartedPayload(projectID, ""))

	c.JSON(http.StatusOK, models.ProcessResponse{
		ProjectID: projectID.String(),
		Status:    "processing",
		EditID:    "", // No editID returned from API
	})
}
