package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"instant-hdr-backend/internal/models"
)

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ok",
	})
}

