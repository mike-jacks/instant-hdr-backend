package supabase

import (
	"github.com/supabase-community/supabase-go"
	"instant-hdr-backend/internal/config"
)

type Client struct {
	Supabase *supabase.Client
	Config   *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	client, err := supabase.NewClient(cfg.SupabaseURL, cfg.SupabasePublishableKey, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		Supabase: client,
		Config:   cfg,
	}, nil
}
