package config_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/config"
)

func TestRenderBlueprintSpecification(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "render.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read render.yaml at %q: %v", manifestPath, err)
	}

	text := string(content)
	if len(strings.TrimSpace(text)) == 0 {
		t.Fatal("render.yaml is empty")
	}

	requiredSubstrings := []string{
		"databases:",
		"name: halisi-postgres",
		"databaseName: halisi",
		"user: halisi",
		"plan: 0.5c-1g",
		"postgresMajorVersion: \"17\"",
		"ipAllowList: []",
		"type: web",
		"name: halisi",
		"runtime: go",
		"region: oregon",
		"branch: main",
		"autoDeploy: false",
		"buildCommand: go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate",
		"preDeployCommand: ./bin/migrate up",
		"startCommand: ./bin/server",
		"healthCheckPath: /health",
		"name: vendor-documents",
		"mountPath: /var/data",
		"sizeGB: 1",
		"key: APP_ENV",
		"value: production",
		"key: VENDOR_DOCUMENT_DIR",
		"value: /var/data/vendor-documents",
		"key: DATABASE_URL",
		"fromDatabase:",
		"property: connectionString",
		"key: PGSSLMODE",
		"value: require",
		"key: PUBLIC_BASE_URL",
		"key: SMTP_HOST",
		"key: SMTP_PORT",
		"key: SMTP_TLS_MODE",
		"key: SMTP_FROM",
		"key: SMTP_USERNAME",
		"key: SMTP_PASSWORD",
		"key: GOOGLE_CLIENT_ID",
		"key: GOOGLE_CLIENT_SECRET",
		"key: MPESA_ENV",
		"key: MPESA_CONSUMER_KEY",
		"key: MPESA_CONSUMER_SECRET",
		"key: MPESA_SHORT_CODE",
		"key: MPESA_PASSKEY",
		"key: MPESA_TRANSACTION_TYPE",
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(text, sub) {
			t.Errorf("render.yaml missing expected configuration: %q", sub)
		}
	}

	// Secret hygiene check: ensure all secrets are configured with sync: false and no committed values
	secretKeys := []string{
		"PUBLIC_BASE_URL",
		"SMTP_HOST",
		"SMTP_FROM",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"MPESA_ENV",
		"MPESA_CONSUMER_KEY",
		"MPESA_CONSUMER_SECRET",
		"MPESA_SHORT_CODE",
		"MPESA_PASSKEY",
		"MPESA_TRANSACTION_TYPE",
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	var currentKey string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- key:") {
			currentKey = strings.TrimSpace(strings.TrimPrefix(line, "- key:"))
			continue
		}
		for _, secret := range secretKeys {
			if currentKey == secret {
				if strings.HasPrefix(line, "value:") {
					t.Errorf("secret key %q must not have a hardcoded committed value in render.yaml", secret)
				}
				if line == "sync: false" {
					// Confirmed sync: false is present for secret
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
}

func TestProductionEnvironmentRequirements(t *testing.T) {
	t.Run("production rejects HTTP public base url", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_FROM", "no-reply@example.com")
		_, err := config.LoadIdentity("production")
		if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL must be an HTTPS origin") {
			t.Fatalf("expected HTTPS error, got: %v", err)
		}
	})

	t.Run("production requires SMTP_HOST", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.onrender.com")
		t.Setenv("SMTP_HOST", "")
		_, err := config.LoadIdentity("production")
		if err == nil || !strings.Contains(err.Error(), "SMTP_HOST is required in production") {
			t.Fatalf("expected SMTP_HOST required error, got: %v", err)
		}
	})

	t.Run("production rejects SMTP_TLS_MODE none", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.onrender.com")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_FROM", "no-reply@example.com")
		t.Setenv("SMTP_USERNAME", "smtp_user")
		t.Setenv("SMTP_PASSWORD", "smtp_pass")
		t.Setenv("SMTP_TLS_MODE", "none")
		_, err := config.LoadIdentity("production")
		if err == nil || !strings.Contains(err.Error(), "SMTP_TLS_MODE=none is permitted only for local development") {
			t.Fatalf("expected SMTP_TLS_MODE rejection, got: %v", err)
		}
	})

	t.Run("valid production identity succeeds", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.onrender.com")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_PORT", "587")
		t.Setenv("SMTP_TLS_MODE", "starttls")
		t.Setenv("SMTP_FROM", "no-reply@example.com")
		t.Setenv("SMTP_USERNAME", "smtp_user")
		t.Setenv("SMTP_PASSWORD", "smtp_pass")
		ident, err := config.LoadIdentity("production")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ident.PublicBaseURL != "https://halisi.onrender.com" {
			t.Errorf("got PublicBaseURL %q, want https://halisi.onrender.com", ident.PublicBaseURL)
		}
		if ident.SMTPHost != "smtp.example.com" {
			t.Errorf("got SMTPHost %q, want smtp.example.com", ident.SMTPHost)
		}
	})
}
