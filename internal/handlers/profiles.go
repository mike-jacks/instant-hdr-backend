package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
)

type ProfilesHandler struct {
	imagenClient *imagen.Client
}

func NewProfilesHandler(imagenClient *imagen.Client) *ProfilesHandler {
	return &ProfilesHandler{
		imagenClient: imagenClient,
	}
}

// GetProfiles godoc
// @Summary     Get user profiles
// @Description Returns a list of available editing profiles for the authenticated user. Each profile has a profile_key (integer) that should be used in process requests. Use the profile_key value when calling the /projects/{project_id}/process endpoint.
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Success     200 {array} object "Array of profile objects. Each object contains: profile_key (int), profile_name (string), profile_type (string: Personal/Talent/Shared), image_type (string: RAW/JPG)"
// @Failure     401 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /profiles [get]
func (h *ProfilesHandler) GetProfiles(c *gin.Context) {
	// Verify user is authenticated (middleware handles this, but we check anyway)
	_, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	profiles, err := h.imagenClient.GetUserProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get profiles",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profiles)
}

