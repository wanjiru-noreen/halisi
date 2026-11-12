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
