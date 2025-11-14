package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type StatusHandler struct {
	dbClient *supabase.DatabaseClient
}

func NewStatusHandler(dbClient *supabase.DatabaseClient) *StatusHandler {
	return &StatusHandler{
		dbClient: dbClient,
	}
}

// GetStatus godoc
// @Summary     Get project status
// @Description Returns the current status and progress of a project. For real-time updates, connect to Supabase Realtime.
// @Tags        status
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       project_id path string true "Project ID (UUID)"
// @Success     200 {object} models.StatusResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /projects/{project_id}/status [get]
func (h *StatusHandler) GetStatus(c *gin.Context) {
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

	project, err := h.dbClient.GetProject(projectID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "project not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.StatusResponse{
		ProjectID: projectID.String(),
		Status:    project.Status,
		Progress:  project.Progress,
		UpdatedAt: project.UpdatedAt,
	})
}

