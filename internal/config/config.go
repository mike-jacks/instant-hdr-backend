package config

import (
	"fmt"
	"os"
)

type Config struct {
	// Imagen API
	ImagenAPIKey        string
	ImagenAPIBaseURL    string
	ImagenWebhookSecret string

	// Supabase
	SupabaseURL            string
	SupabasePublishableKey string
	SupabaseJWTSecret      string
	SupabaseStorageBucket  string

	// Webhook
	WebhookCallbackURL string

	// Database
	DatabaseURL string

	// Server
	Port        string
	Environment string
	BaseURL     string
}

func Load() (*Config, error) {
	cfg := &Config{
		ImagenAPIKey:        getEnv("IMAGEN_API_KEY", ""),
		ImagenAPIBaseURL:    getEnv("IMAGEN_API_BASE_URL", "https://api.imagen-ai.com/v1/"),
		ImagenWebhookSecret: getEnv("IMAGEN_WEBHOOK_SECRET", ""),

		SupabaseURL:            getEnv("SUPABASE_URL", ""),
		SupabasePublishableKey: getEnv("SUPABASE_PUBLISHABLE_KEY", ""),
		SupabaseJWTSecret:      getEnv("SUPABASE_JWT_SECRET", ""),
		SupabaseStorageBucket:  getEnv("SUPABASE_STORAGE_BUCKET", "processed-images"),

		WebhookCallbackURL: getEnv("WEBHOOK_CALLBACK_URL", ""),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ImagenAPIKey == "" {
		return fmt.Errorf("IMAGEN_API_KEY is required")
	}
	if c.SupabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	if c.SupabasePublishableKey == "" {
		return fmt.Errorf("SUPABASE_PUBLISHABLE_KEY is required")
	}
	if c.SupabaseJWTSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
