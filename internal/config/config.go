// Package config loads HomeBox MCP server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds everything needed to reach and authenticate against a Homebox instance.
type Config struct {
	// URL is the base URL of the Homebox instance, e.g. "https://homebox.example.com".
	URL string
	// Email is the login username (Homebox uses email as username).
	Email string
	// Password is the login password.
	Password string
}

// Load reads configuration from the environment. It fails fast with a
// descriptive error listing every missing variable at once.
func Load() (*Config, error) {
	cfg := &Config{
		URL:      strings.TrimRight(os.Getenv("HOMEBOX_URL"), "/"),
		Email:    os.Getenv("HOMEBOX_EMAIL"),
		Password: os.Getenv("HOMEBOX_PASSWORD"),
	}

	var missing []string
	if cfg.URL == "" {
		missing = append(missing, "HOMEBOX_URL")
	}
	if cfg.Email == "" {
		missing = append(missing, "HOMEBOX_EMAIL")
	}
	if cfg.Password == "" {
		missing = append(missing, "HOMEBOX_PASSWORD")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
		return nil, fmt.Errorf("HOMEBOX_URL must start with http:// or https://, got %q", cfg.URL)
	}
	return cfg, nil
}
