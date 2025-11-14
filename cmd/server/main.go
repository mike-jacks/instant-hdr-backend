// @title           Instant HDR Backend API
// @version         1.0.0
// @description     Backend API for processing bracketed images with Imagen AI HDR merging. This API handles project creation, image uploads, HDR processing, and real-time status updates via Supabase Realtime.

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"log"
	"net/http"

	"instant-hdr-backend/docs"
	"instant-hdr-backend/internal/config"
	"instant-hdr-backend/internal/database"
	"instant-hdr-backend/internal/handlers"
	"instant-hdr-backend/internal/imagen"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/services"
	"instant-hdr-backend/internal/supabase"
	"net/url"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Update Swagger docs with dynamic base URL
	if cfg.BaseURL != "" {
		baseURL, err := url.Parse(cfg.BaseURL)
		if err == nil {
			// Extract host (e.g., "localhost:8080" or "your-app.railway.app")
			docs.SwaggerInfo.Host = baseURL.Host
			// Set scheme based on URL
			if baseURL.Scheme == "https" {
				docs.SwaggerInfo.Schemes = []string{"https", "http"}
			} else {
				docs.SwaggerInfo.Schemes = []string{"http", "https"}
			}
		}
	}

	// Database connection string
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Println("Warning: DATABASE_URL not set. Migrations will be skipped.")
		log.Println("Please set DATABASE_URL environment variable with your Supabase PostgreSQL connection string")
	}

	// Create database client (we'll use Supabase PostgREST for now, but need direct DB for migrations)
	// For migrations, we need direct PostgreSQL connection
	// This is a simplified version - in production, you'd have proper connection string management

	// Initialize Imagen client
	imagenClient := imagen.NewClient(cfg.ImagenAPIBaseURL, cfg.ImagenAPIKey)

	// Initialize Supabase clients
	supabaseClient, err := supabase.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Supabase client: %v", err)
	}

	storageClient, err := supabase.NewStorageClient(cfg.SupabaseURL, cfg.SupabasePublishableKey, cfg.SupabaseStorageBucket)
	if err != nil {
		log.Fatalf("Failed to initialize storage client: %v", err)
	}

	realtimeClient := supabase.NewRealtimeClient(supabaseClient.Supabase)

	// Create database client for direct queries
	var dbClient *supabase.DatabaseClient
	if dbURL != "" {
		var err error
		dbClient, err = supabase.NewDatabaseClient(dbURL)
		if err != nil {
			log.Printf("Warning: Failed to initialize database client: %v", err)
			log.Println("Database operations will be limited. Please configure DATABASE_URL properly.")
		} else {
			defer dbClient.Close()

			// Run migrations
			migrator, err := database.NewMigrator(dbURL)
			if err != nil {
				log.Printf("Warning: Failed to initialize migrator: %v", err)
			} else {
				defer migrator.Close()
				if err := migrator.Run(); err != nil {
					log.Printf("Warning: Migration failed: %v", err)
				} else {
					log.Println("Migrations completed successfully")
				}
			}
		}
	}

	// Initialize storage service (only if dbClient is available)
	var storageService *services.StorageService
	if dbClient != nil {
		storageService = services.NewStorageService(imagenClient, dbClient, storageClient, realtimeClient)
	}

	// Initialize handlers (dbClient might be nil, handlers should handle this)
	projectsHandler := handlers.NewProjectsHandler(imagenClient, dbClient, storageClient)
	uploadHandler := handlers.NewUploadHandler(imagenClient, dbClient, realtimeClient)
	processHandler := handlers.NewProcessHandler(imagenClient, dbClient, realtimeClient, cfg.WebhookCallbackURL)
	statusHandler := handlers.NewStatusHandler(dbClient)
	filesHandler := handlers.NewFilesHandler(dbClient)
	profilesHandler := handlers.NewProfilesHandler(imagenClient)

	// Webhook handler requires storage service
	if storageService == nil {
		log.Println("Warning: Storage service not available. Webhook handler will not work properly.")
		// Create a nil-safe storage service or handle this differently
	}
	webhookHandler := handlers.NewWebhookHandler(cfg, storageService)

	// Setup router
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check (no auth)
	router.GET("/health", handlers.HealthHandler)

	// API routes
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg))

	// Project routes
	api.POST("/projects", projectsHandler.CreateProject)
	api.GET("/projects", projectsHandler.ListProjects)
	api.GET("/projects/:project_id", projectsHandler.GetProject)
	api.DELETE("/projects/:project_id", projectsHandler.DeleteProject)

	// Upload and processing
	api.POST("/projects/:project_id/upload", uploadHandler.Upload)
	api.POST("/projects/:project_id/process", processHandler.Process)

	// Status and files
	api.GET("/projects/:project_id/status", statusHandler.GetStatus)
	api.GET("/projects/:project_id/files", filesHandler.GetFiles)

	// Profiles
	api.GET("/profiles", profilesHandler.GetProfiles)

	// Webhook (no auth, uses HMAC)
	router.POST("/api/v1/webhooks/imagen", webhookHandler.HandleWebhook)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
