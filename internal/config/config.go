package config

import (
	"fmt"
	"os"
)

type Config struct {
	// AutoEnhance AI API
	AutoEnhanceAPIKey       string
	AutoEnhanceAPIBaseURL   string
	AutoEnhanceWebhookToken string

	// Imagen API (kept for backward compatibility, not used)
	ImagenAPIKey        string
	ImagenAPIBaseURL    string
	ImagenWebhookSecret string

	// Supabase
	SupabaseURL            string
	SupabasePublishableKey string
	SupabaseServiceRoleKey string
	SupabaseUseRLS         bool   // If true, use publishable key + RLS; if false, use service role key
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
		// AutoEnhance AI API
		AutoEnhanceAPIKey:       getEnv("AUTOENHANCE_API_KEY", ""),
		AutoEnhanceAPIBaseURL:   getEnv("AUTOENHANCE_API_BASE_URL", "https://api.autoenhance.ai"),
		AutoEnhanceWebhookToken: getEnv("AUTOENHANCE_WEBHOOK_TOKEN", ""),

		// Imagen API (kept for backward compatibility, not used)
		ImagenAPIKey:        getEnv("IMAGEN_API_KEY", ""),
		ImagenAPIBaseURL:    getEnv("IMAGEN_API_BASE_URL", "https://api.imagen-ai.com/v1/"),
		ImagenWebhookSecret: getEnv("IMAGEN_WEBHOOK_SECRET", ""),

		SupabaseURL:            getEnv("SUPABASE_URL", ""),
		SupabasePublishableKey: getEnv("SUPABASE_PUBLISHABLE_KEY", ""),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseUseRLS:         getEnv("SUPABASE_USE_RLS", "true") == "true", // Default to RLS (more secure)
		SupabaseJWTSecret:      getEnv("SUPABASE_JWT_SECRET", ""),
		SupabaseStorageBucket:  getEnv("SUPABASE_STORAGE_BUCKET", "hdr-images"),

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
	// AutoEnhance AI API is required
	if c.AutoEnhanceAPIKey == "" {
		return fmt.Errorf("AUTOENHANCE_API_KEY is required")
	}
	if c.AutoEnhanceAPIBaseURL == "" {
		return fmt.Errorf("AUTOENHANCE_API_BASE_URL is required")
	}

	// Supabase is required
	if c.SupabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	if c.SupabasePublishableKey == "" {
		return fmt.Errorf("SUPABASE_PUBLISHABLE_KEY is required")
	}
	
	// Service role key is only required if NOT using RLS
	if !c.SupabaseUseRLS && c.SupabaseServiceRoleKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY is required when SUPABASE_USE_RLS=false")
	}
	if c.SupabaseJWTSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}

	// Imagen API fields are kept for backward compatibility but not validated
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
