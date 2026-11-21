package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProductionSettings encapsulates validated production settings.
type ProductionSettings struct {
	Config       Config
	Identity     Identity
	MPesa        *MPesa
	VendorDocDir string
}

// ValidateProduction verifies that all required environment variables, secrets,
// and domain configurations for running in production are set, well-formed,
// and meet production security constraints. It never returns sensitive credential values.
func ValidateProduction() (ProductionSettings, error) {
	cfg, err := Load()
	if err != nil {
		return ProductionSettings{}, fmt.Errorf("configuration error: %w", err)
	}
	if cfg.AppEnv != "production" {
		return ProductionSettings{}, fmt.Errorf("APP_ENV must be production (got %q)", cfg.AppEnv)
	}
	databaseURL, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		return ProductionSettings{}, fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	sslmode := databaseURL.Query().Get("sslmode")
	pgSSLMode := os.Getenv("PGSSLMODE")
	if !isSecureSSLMode(sslmode) && !isSecureSSLMode(pgSSLMode) {
		return ProductionSettings{}, fmt.Errorf("DATABASE_URL or PGSSLMODE must require TLS in production (e.g. sslmode=require)")
	}
	docDir := os.Getenv("VENDOR_DOCUMENT_DIR")
	cleanDocDir := filepath.ToSlash(filepath.Clean(docDir))
	if docDir == "" || !(strings.HasPrefix(cleanDocDir, "/var/data/") || cleanDocDir == "/var/data") {
		return ProductionSettings{}, fmt.Errorf("VENDOR_DOCUMENT_DIR must be located within persistent mount /var/data in production")
	}
	ident, err := LoadIdentity("production")
	if err != nil {
		return ProductionSettings{}, fmt.Errorf("identity error: %w", err)
	}
	if ident.SMTPUsername == "" || ident.SMTPPassword == "" {
		return ProductionSettings{}, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD are required in production")
	}
	mpesa, err := LoadMPesa()
	if err != nil {
		return ProductionSettings{}, fmt.Errorf("M-Pesa error: %w", err)
	}
	return ProductionSettings{
		Config:       cfg,
		Identity:     ident,
		MPesa:        mpesa,
		VendorDocDir: docDir,
	}, nil
}

func isSecureSSLMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

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
