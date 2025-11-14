package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"instant-hdr-backend/internal/config"
	"instant-hdr-backend/internal/middleware"
)

func TestAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
	}

	router := gin.New()
	router.Use(middleware.AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
	}

	router := gin.New()
	router.Use(middleware.AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret-key-for-jwt-signing-must-be-long-enough",
	}

	// Create a valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
	})
	tokenString, _ := token.SignedString([]byte(cfg.SupabaseJWTSecret))

	router := gin.New()
	router.Use(middleware.AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get(middleware.UserIDKey)
		assert.True(t, exists)
		assert.Equal(t, "user-123", userID)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

