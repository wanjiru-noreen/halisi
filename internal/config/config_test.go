package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	const databaseURL = "postgres://example_user:example_password@localhost:5432/example_db"
	tests := []struct {
		name, port, databaseURL, appEnv string
		want                            Config
		wantError                       string
	}{
		{name: "defaults", databaseURL: databaseURL, want: Config{"8080", databaseURL, "development"}},
		{name: "explicit values", port: "9090", databaseURL: databaseURL, appEnv: "production", want: Config{"9090", databaseURL, "production"}},
		{name: "test environment", port: "1", databaseURL: "postgresql://localhost/test", appEnv: "test", want: Config{"1", "postgresql://localhost/test", "test"}},
		{name: "maximum port", port: "65535", databaseURL: databaseURL, want: Config{"65535", databaseURL, "development"}},
		{name: "trim whitespace", port: " 8081 ", databaseURL: " " + databaseURL + " ", appEnv: " test ", want: Config{"8081", databaseURL, "test"}},
		{name: "missing database", wantError: "DATABASE_URL is required"},
		{name: "blank database", databaseURL: " \t", wantError: "DATABASE_URL is required"},
		{name: "invalid port", port: "abc", databaseURL: databaseURL, wantError: "PORT"},
		{name: "zero port", port: "0", databaseURL: databaseURL, wantError: "PORT"},
		{name: "negative port", port: "-1", databaseURL: databaseURL, wantError: "PORT"},
		{name: "large port", port: "65536", databaseURL: databaseURL, wantError: "PORT"},
		{name: "invalid environment", databaseURL: databaseURL, appEnv: "invalid", wantError: "APP_ENV"},
		{name: "invalid scheme", databaseURL: "https://localhost/db", wantError: "DATABASE_URL"},
		{name: "missing host", databaseURL: "postgres:///db", wantError: "DATABASE_URL"},
		{name: "malformed URL", databaseURL: "postgres://example_user:example_password%zz@localhost/db", wantError: "DATABASE_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("APP_ENV", tt.appEnv)
			got, err := Load()
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("Load() error = %v, want error containing %q", err, tt.wantError)
				}
				if strings.Contains(err.Error(), "example_password") {
					t.Fatal("error exposes database credentials")
				}
				if got != (Config{}) {
					t.Fatal("Load() returned partial configuration on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got != tt.want {
				t.Fatal("Load() did not return the expected configuration")
			}
		})
	}
}

func TestPort(t *testing.T) {
	for _, port := range []string{"", "9090"} {
		t.Run("PORT="+port, func(t *testing.T) {
			t.Setenv("PORT", port)
			want := port
			if want == "" {
				want = "8080"
			}
			if got := Port(); got != want {
				t.Fatalf("Port() = %q, want %q", got, want)
			}
		})
	}
}

func TestValidateProduction(t *testing.T) {
	setBaseValidProductionEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("APP_ENV", "production")
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://prod_user:secret_password@db.halisi.internal:5432/halisi_prod?sslmode=require")
		t.Setenv("VENDOR_DOCUMENT_DIR", "/var/data/vendor-documents")
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.onrender.com")
		t.Setenv("SMTP_HOST", "smtp.sendgrid.net")
		t.Setenv("SMTP_PORT", "587")
		t.Setenv("SMTP_TLS_MODE", "starttls")
		t.Setenv("SMTP_FROM", "no-reply@halisi.co.ke")
		t.Setenv("SMTP_USERNAME", "apikey")
		t.Setenv("SMTP_PASSWORD", "sg_secret_key")
		t.Setenv("GOOGLE_CLIENT_ID", "")
		t.Setenv("GOOGLE_CLIENT_SECRET", "")
		t.Setenv("MPESA_ENV", "")
		t.Setenv("MPESA_CONSUMER_KEY", "")
		t.Setenv("MPESA_CONSUMER_SECRET", "")
		t.Setenv("MPESA_SHORT_CODE", "")
		t.Setenv("MPESA_PASSKEY", "")
		t.Setenv("MPESA_TRANSACTION_TYPE", "")
	}

	t.Run("valid complete production environment", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		settings, err := ValidateProduction()
		if err != nil {
			t.Fatalf("expected valid production settings, got error: %v", err)
		}
		if settings.Config.AppEnv != "production" {
			t.Errorf("expected APP_ENV production, got %q", settings.Config.AppEnv)
		}
		if settings.Identity.PublicBaseURL != "https://halisi.onrender.com" {
			t.Errorf("expected PublicBaseURL https://halisi.onrender.com, got %q", settings.Identity.PublicBaseURL)
		}
		if settings.VendorDocDir != "/var/data/vendor-documents" {
			t.Errorf("expected VendorDocDir /var/data/vendor-documents, got %q", settings.VendorDocDir)
		}
	})

	t.Run("valid production with custom domain", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "https://halisi.co.ke")
		settings, err := ValidateProduction()
		if err != nil {
			t.Fatalf("expected valid custom domain, got: %v", err)
		}
		if settings.Identity.PublicBaseURL != "https://halisi.co.ke" {
			t.Errorf("expected PublicBaseURL https://halisi.co.ke, got %q", settings.Identity.PublicBaseURL)
		}
	})

	t.Run("rejects non-production environment", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("APP_ENV", "development")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "APP_ENV must be production") {
			t.Fatalf("expected APP_ENV error, got: %v", err)
		}
	})

	t.Run("rejects missing or non-persistent VENDOR_DOCUMENT_DIR", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		for _, invalidDir := range []string{"", "relative/path", "/tmp/vendor-documents", "/etc/data"} {
			t.Setenv("VENDOR_DOCUMENT_DIR", invalidDir)
			_, err := ValidateProduction()
			if err == nil || !strings.Contains(err.Error(), "VENDOR_DOCUMENT_DIR must be located within persistent mount /var/data") {
				t.Fatalf("expected persistent mount error for %q, got: %v", invalidDir, err)
			}
		}
	})

	t.Run("rejects insecure HTTP PUBLIC_BASE_URL in production", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("PUBLIC_BASE_URL", "http://halisi.co.ke")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL must be an HTTPS origin") {
			t.Fatalf("expected HTTPS origin error, got: %v", err)
		}
	})

	t.Run("rejects unset SMTP credentials in production", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("SMTP_USERNAME", "")
		t.Setenv("SMTP_PASSWORD", "")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "SMTP_USERNAME and SMTP_PASSWORD are required in production") {
			t.Fatalf("expected SMTP credentials error, got: %v", err)
		}
	})

	t.Run("rejects partial Google OAuth credentials", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "google-client-id-only")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must both be configured") {
			t.Fatalf("expected Google OAuth error, got: %v", err)
		}
	})

	t.Run("rejects partial M-Pesa credentials", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("MPESA_ENV", "production")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "M-Pesa error") {
			t.Fatalf("expected M-Pesa error, got: %v", err)
		}
	})

	t.Run("rejects non-TLS database URL in production", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("DATABASE_URL", "postgres://user:secret@db.internal:5432/halisi")
		t.Setenv("PGSSLMODE", "")
		_, err := ValidateProduction()
		if err == nil || !strings.Contains(err.Error(), "DATABASE_URL or PGSSLMODE must require TLS in production") {
			t.Fatalf("expected TLS error, got: %v", err)
		}
	})

	t.Run("allows unadorned database URL when PGSSLMODE=require is set", func(t *testing.T) {
		setBaseValidProductionEnv(t)
		t.Setenv("DATABASE_URL", "postgres://user:secret@db.internal:5432/halisi")
		t.Setenv("PGSSLMODE", "require")
		_, err := ValidateProduction()
		if err != nil {
			t.Fatalf("expected valid with PGSSLMODE=require, got: %v", err)
		}
	})
}
