package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"instant-hdr-backend/internal/models"
)

// HealthHandler godoc
// @Summary     Health check
// @Description Returns the health status of the API
// @Tags        health
// @Accept      json
// @Produce     json
// @Success     200 {object} models.HealthResponse
// @Router      /health [get]
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ok",
	})
}

