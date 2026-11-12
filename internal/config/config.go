package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config contains validated application settings loaded from the environment.
// Avoid logging Config because DatabaseURL may contain credentials.
type Config struct {
	Port        string
	DatabaseURL string
	AppEnv      string
}

// Load reads environment variables. DATABASE_URL is required; PORT and APP_ENV
// default to 8080 and development. It never includes secret values in errors.
func Load() (Config, error) {
	cfg := Config{
		Port:        Port(),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AppEnv:      strings.TrimSpace(os.Getenv("APP_ENV")),
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}
	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	databaseURL, err := url.Parse(cfg.DatabaseURL)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") || databaseURL.Hostname() == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with a host")
	}
	switch cfg.AppEnv {
	case "development", "test", "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	return cfg, nil
}

// Port returns the configured HTTP port, defaulting to 8080.
func Port() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return port
	}
	return "8080"
}
