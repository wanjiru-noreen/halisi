package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/config"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/routes"
)

type mockPinger struct {
	err error
}

func (m mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestReadiness_SecurityHeaders(t *testing.T) {
	handler := routes.New(mockPinger{err: nil})

	t.Run("standard security headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", rec.Code)
		}

		headers := map[string]string{
			"Content-Security-Policy": "default-src 'self'",
			"X-Frame-Options":         "DENY",
			"X-Content-Type-Options":  "nosniff",
			"Referrer-Policy":         "same-origin",
		}

		for header, expectedSubstring := range headers {
			val := rec.Header().Get(header)
			if val == "" {
				t.Errorf("missing required security header %q", header)
			} else if !strings.Contains(val, expectedSubstring) {
				t.Errorf("header %q = %q, expected to contain %q", header, val, expectedSubstring)
			}
		}
	})

	t.Run("HSTS header present on HTTPS / forwarded HTTPS requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		hsts := rec.Header().Get("Strict-Transport-Security")
		if hsts == "" {
			t.Fatal("missing Strict-Transport-Security on HTTPS request")
		}
		if !strings.Contains(hsts, "max-age=") {
			t.Errorf("unexpected HSTS header value: %q", hsts)
		}
	})
}

func TestReadiness_HealthProbes(t *testing.T) {
	t.Run("liveness probe /health returns 200 ok", func(t *testing.T) {
		handler := routes.New(nil)
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != `{"status":"ok"}` {
			t.Fatalf("expected JSON body {\"status\":\"ok\"}, got %q", body)
		}
	})

	t.Run("database probe /health/db returns 200 ok when healthy", func(t *testing.T) {
		handler := routes.New(mockPinger{err: nil})
		req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != `{"status":"ok"}` {
			t.Fatalf("expected JSON body {\"status\":\"ok\"}, got %q", body)
		}
	})

	t.Run("database probe /health/db returns 503 unavailable when ping fails without leaking credentials", func(t *testing.T) {
		secretErr := errors.New("driver connection error containing postgres://user:secretpassword@host")
		handler := routes.New(mockPinger{err: secretErr})
		req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
		}
		body := rec.Body.String()
		if body != `{"status":"unavailable"}` {
			t.Fatalf("expected {\"status\":\"unavailable\"}, got %q", body)
		}
		if strings.Contains(body, "secretpassword") {
			t.Fatal("health check leaked credentials in body")
		}
	})
}

func TestReadiness_ProductionEnvironmentValidation(t *testing.T) {
	setValidProductionEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("APP_ENV", "production")
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://halisi:secret@db:5432/halisi?sslmode=require")
		t.Setenv("VENDOR_DOCUMENT_DIR", "/var/data/vendor-documents")
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.onrender.com")
		t.Setenv("SMTP_HOST", "smtp.sendgrid.net")
		t.Setenv("SMTP_PORT", "587")
		t.Setenv("SMTP_TLS_MODE", "starttls")
		t.Setenv("SMTP_FROM", "no-reply@halisi.co.ke")
		t.Setenv("SMTP_USERNAME", "apikey")
		t.Setenv("SMTP_PASSWORD", "secret_smtp_key")
		t.Setenv("GOOGLE_CLIENT_ID", "")
		t.Setenv("GOOGLE_CLIENT_SECRET", "")
		t.Setenv("MPESA_ENV", "")
		t.Setenv("MPESA_CONSUMER_KEY", "")
		t.Setenv("MPESA_CONSUMER_SECRET", "")
		t.Setenv("MPESA_SHORT_CODE", "")
		t.Setenv("MPESA_PASSKEY", "")
		t.Setenv("MPESA_TRANSACTION_TYPE", "")
	}

	t.Run("valid production settings pass validation", func(t *testing.T) {
		setValidProductionEnv(t)
		settings, err := config.ValidateProduction()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
		if settings.Config.AppEnv != "production" {
			t.Errorf("expected APP_ENV production, got %q", settings.Config.AppEnv)
		}
		if settings.Identity.PublicBaseURL != "https://halisi.onrender.com" {
			t.Errorf("expected PublicBaseURL https://halisi.onrender.com, got %q", settings.Identity.PublicBaseURL)
		}
	})

	t.Run("rejects non-production APP_ENV", func(t *testing.T) {
		setValidProductionEnv(t)
		t.Setenv("APP_ENV", "development")
		_, err := config.ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "APP_ENV must be production") {
			t.Fatalf("expected APP_ENV error, got: %v", err)
		}
	})

	t.Run("rejects non-HTTPS PUBLIC_BASE_URL", func(t *testing.T) {
		setValidProductionEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "http://insecure.example.com")
		_, err := config.ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL must be an HTTPS origin") {
			t.Fatalf("expected HTTPS error, got: %v", err)
		}
	})

	t.Run("rejects unencrypted SMTP in production", func(t *testing.T) {
		setValidProductionEnv(t)
		t.Setenv("SMTP_TLS_MODE", "none")
		_, err := config.ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "SMTP_TLS_MODE=none is permitted only for local development") {
			t.Fatalf("expected SMTP TLS mode error, got: %v", err)
		}
	})
}

func TestReadiness_BlueprintSpecification(t *testing.T) {
	renderYAMLPath := filepath.Join("..", "..", "render.yaml")
	content, err := os.ReadFile(renderYAMLPath)
	if err != nil {
		t.Fatalf("could not read render.yaml at %q: %v", renderYAMLPath, err)
	}

	text := string(content)
	checks := []string{
		"databases:",
		"name: halisi-postgres",
		"postgresMajorVersion: \"17\"",
		"type: web",
		"name: halisi",
		"runtime: go",
		"plan: starter",
		"branch: main",
		"buildCommand: go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate",
		"preDeployCommand: ./bin/migrate up",
		"startCommand: ./bin/server",
		"healthCheckPath: /health",
		"mountPath: /var/data",
		"name: vendor-documents",
		"fromDatabase:",
		"property: connectionString",
	}

	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Errorf("render.yaml missing readiness requirement: %q", check)
		}
	}
}

func TestReadiness_LiveDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("TEST_MIGRATION_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run live database readiness tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("live database connection failed: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("live database ping failed: %v", err)
	}
}
