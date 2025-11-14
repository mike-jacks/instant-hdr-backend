package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
)

type ProfilesHandler struct {
	// AutoEnhance doesn't have profiles - using enhance_type instead
}

func NewProfilesHandler() *ProfilesHandler {
	return &ProfilesHandler{}
}

// GetProfiles godoc
// @Summary     Get enhancement types
// @Description AutoEnhance AI doesn't use profiles. Returns available enhance_type options instead. Use enhance_type in process requests: "property", "property_usa", "warm", "neutral", "modern"
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Success     200 {array} object "Array of enhancement type options"
// @Failure     401 {object} models.ErrorResponse
// @Router      /profiles [get]
func (h *ProfilesHandler) GetProfiles(c *gin.Context) {
	// Verify user is authenticated (middleware handles this, but we check anyway)
	_, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	// AutoEnhance doesn't have profiles - return enhance_type options
	enhanceTypes := []map[string]interface{}{
		{"enhance_type": "property", "description": "Property enhancement (default for real estate)"},
		{"enhance_type": "property_usa", "description": "Property enhancement (USA style)"},
		{"enhance_type": "warm", "description": "Warm enhancement"},
		{"enhance_type": "neutral", "description": "Neutral enhancement"},
		{"enhance_type": "modern", "description": "Modern enhancement"},
	}

	c.JSON(http.StatusOK, enhanceTypes)
}
