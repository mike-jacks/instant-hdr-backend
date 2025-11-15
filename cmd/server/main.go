// @title           Instant HDR Backend API
// @version         1.0.0
// @description     Backend API for processing bracketed images with AutoEnhance AI HDR merging. This API handles order creation, image uploads, HDR processing, and real-time status updates via Supabase Realtime.

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
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/config"
	"instant-hdr-backend/internal/database"
	"instant-hdr-backend/internal/handlers"
	_ "instant-hdr-backend/internal/imagen" // Kept for reference, not used
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

	// Initialize AutoEnhance AI client
	autoenhanceClient := autoenhance.NewClient(cfg.AutoEnhanceAPIBaseURL, cfg.AutoEnhanceAPIKey)

	// Imagen client kept for reference but not used
	// imagenClient := imagen.NewClient(cfg.ImagenAPIBaseURL, cfg.ImagenAPIKey)

	// Initialize Supabase clients
	supabaseClient, err := supabase.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Supabase client: %v", err)
	}

	// Storage client: Choose between RLS (publishable key) or service role key based on config
	var storageKey string
	if cfg.SupabaseUseRLS {
		log.Println("Using Supabase Storage with RLS (publishable key) - More secure")
		storageKey = cfg.SupabasePublishableKey
	} else {
		log.Println("Using Supabase Storage with Service Role Key - Bypasses RLS")
		storageKey = cfg.SupabaseServiceRoleKey
	}
	storageClient, err := supabase.NewStorageClient(cfg.SupabaseURL, storageKey, cfg.SupabaseStorageBucket)
	if err != nil {
		log.Fatalf("Failed to initialize storage client: %v", err)
	}

	// Use service role key for Realtime (server-side publishing)
	// Service role key bypasses RLS and is required for server-side broadcast
	if cfg.SupabaseServiceRoleKey == "" {
		log.Fatalf("SUPABASE_SERVICE_ROLE_KEY is required for Realtime broadcast")
	}
	realtimeClient := supabase.NewRealtimeClient(supabaseClient.Supabase, cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)

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
		storageService = services.NewStorageService(autoenhanceClient, dbClient, storageClient, realtimeClient)
	}

	// Initialize handlers (dbClient might be nil, handlers should handle this)
	ordersHandler := handlers.NewOrdersHandler(autoenhanceClient, dbClient, storageClient)
	uploadHandler := handlers.NewUploadHandler(autoenhanceClient, dbClient, realtimeClient)
	processHandler := handlers.NewProcessHandler(autoenhanceClient, dbClient, realtimeClient)
	statusHandler := handlers.NewStatusHandler(dbClient, autoenhanceClient)
	filesHandler := handlers.NewFilesHandler(dbClient, autoenhanceClient)
	imagesHandler := handlers.NewImagesHandler(autoenhanceClient, dbClient, storageClient)

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

	// Health check (no auth) - available at root level
	router.GET("/health", handlers.HealthHandler)

	// API routes - public endpoints (no auth)
	apiPublic := router.Group("/api/v1")
	apiPublic.GET("/health", handlers.HealthHandler)
	// Webhook endpoint (uses AutoEnhance webhook token, not JWT)
	apiPublic.POST("/webhooks/autoenhance", webhookHandler.HandleWebhook)

	// API routes - protected endpoints (with auth)
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg))

	// Order routes
	api.POST("/orders", ordersHandler.CreateOrder)
	api.GET("/orders", ordersHandler.ListOrders)
	api.GET("/orders/:order_id", ordersHandler.GetOrder)
	api.GET("/orders/:order_id/verify", ordersHandler.VerifyOrderUploads) // Verify uploads with AutoEnhance
	api.DELETE("/orders/:order_id", ordersHandler.DeleteOrder)

	// Upload and processing
	api.POST("/orders/:order_id/upload", uploadHandler.Upload)
	api.POST("/orders/:order_id/process", processHandler.Process)

	// Status and files
	api.GET("/orders/:order_id/status", statusHandler.GetStatus)
	api.GET("/orders/:order_id/files", filesHandler.GetFiles)                        // Processed files only
	api.GET("/orders/:order_id/brackets", filesHandler.GetBrackets)                  // Uploaded brackets (raw images)
	api.DELETE("/orders/:order_id/brackets/:bracket_id", filesHandler.DeleteBracket) // Delete bracket

	// Images - list, download, and delete processed images
	api.GET("/orders/:order_id/images", imagesHandler.ListImages)
	api.POST("/orders/:order_id/images/:image_id/download", imagesHandler.DownloadImage)
	api.DELETE("/orders/:order_id/images/:image_id", imagesHandler.DeleteImage)

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
