package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type ProjectsHandler struct {
	imagenClient   *imagen.Client
	dbClient       *supabase.DatabaseClient
	storageClient  *supabase.StorageClient
}

func NewProjectsHandler(imagenClient *imagen.Client, dbClient *supabase.DatabaseClient, storageClient *supabase.StorageClient) *ProjectsHandler {
	return &ProjectsHandler{
		imagenClient:  imagenClient,
		dbClient:      dbClient,
		storageClient: storageClient,
	}
}

func (h *ProjectsHandler) CreateProject(c *gin.Context) {
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

	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no JSON body, use empty metadata
		req.Metadata = make(map[string]interface{})
	}

	// Create Imagen project with retry
	var imagenProjectUUID string
	err = h.imagenClient.RetryWithBackoff(func() error {
		var err error
		imagenProjectUUID, err = h.imagenClient.CreateProject()
		return err
	}, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create imagen project",
			Message: err.Error(),
		})
		return
	}

	// Create project in database
	project, err := h.dbClient.CreateProject(userID, imagenProjectUUID, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create project",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ProjectResponse{
		ID:                project.ID.String(),
		ImagenProjectUUID: project.ImagenProjectUUID,
		Status:            project.Status,
		Progress:          project.Progress,
		CreatedAt:         project.CreatedAt,
		UpdatedAt:         project.UpdatedAt,
	})
}

func (h *ProjectsHandler) ListProjects(c *gin.Context) {
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

	projects, err := h.dbClient.ListProjects(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list projects",
			Message: err.Error(),
		})
		return
	}

	summaries := make([]models.ProjectSummary, len(projects))
	for i, p := range projects {
		summaries[i] = models.ProjectSummary{
			ID:        p.ID.String(),
			Status:    p.Status,
			Progress:  p.Progress,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, models.ProjectListResponse{Projects: summaries})
}

func (h *ProjectsHandler) GetProject(c *gin.Context) {
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

	var metadata map[string]interface{}
	if len(project.Metadata) > 0 {
		json.Unmarshal(project.Metadata, &metadata)
	}

	response := models.ProjectResponse{
		ID:                project.ID.String(),
		ImagenProjectUUID: project.ImagenProjectUUID,
		Status:            project.Status,
		Progress:          project.Progress,
		Metadata:          metadata,
		CreatedAt:         project.CreatedAt,
		UpdatedAt:         project.UpdatedAt,
	}

	if project.ProfileKey.Valid {
		response.ProfileKey = project.ProfileKey.String
	}
	if project.EditID.Valid {
		response.EditID = project.EditID.String
	}
	if project.ErrorMessage.Valid {
		response.ErrorMessage = project.ErrorMessage.String
	}

	c.JSON(http.StatusOK, response)
}

func (h *ProjectsHandler) DeleteProject(c *gin.Context) {
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

	// Get project to get imagen_project_uuid
	project, err := h.dbClient.GetProject(projectID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "project not found",
			Message: err.Error(),
		})
		return
	}

	// Delete from Imagen with retry
	err = h.imagenClient.RetryWithBackoff(func() error {
		return h.imagenClient.DeleteProject(project.ImagenProjectUUID)
	}, 3)
	if err != nil {
		// Log error but continue with database deletion
	}

	// Delete files from storage
	if err := h.storageClient.DeleteProjectFiles(userID, projectID); err != nil {
		// Log error but continue
	}

	// Delete from database (cascade will delete files)
	if err := h.dbClient.DeleteProject(projectID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to delete project",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "project deleted successfully"})
}

