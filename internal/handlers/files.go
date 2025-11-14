package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type FilesHandler struct {
	dbClient *supabase.DatabaseClient
}

func NewFilesHandler(dbClient *supabase.DatabaseClient) *FilesHandler {
	return &FilesHandler{
		dbClient: dbClient,
	}
}

func (h *FilesHandler) GetFiles(c *gin.Context) {
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

	files, err := h.dbClient.GetProjectFiles(projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get files",
			Message: err.Error(),
		})
		return
	}

	fileResponses := make([]models.FileResponse, len(files))
	for i, file := range files {
		fileResponses[i] = models.FileResponse{
			ID:         file.ID.String(),
			Filename:   file.Filename,
			StorageURL: file.StorageURL,
			FileSize:   file.FileSize.Int64,
			MimeType:   file.MimeType,
			IsFinal:    file.IsFinal,
			CreatedAt:  file.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, models.FilesResponse{Files: fileResponses})
}

