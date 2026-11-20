package main

import (
	"os"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	tests := [][]string{
		{},
		{"invalid"},
		{"up", "extra"},
		{"ping", "extra"},
	}
	for _, args := range tests {
		err := run(args)
		if err == nil || !strings.Contains(err.Error(), "usage: go run ./cmd/migrate [up|down|version|ping]") {
			t.Errorf("run(%v) error = %v, want usage error", args, err)
		}
	}
}

func TestRunPingRequiresDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	err := run([]string{"ping"})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("expected DATABASE_URL error, got: %v", err)
	}
}

func TestRunPingFailsSafely(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test_user:secret_password@127.0.0.1:54329/test_db?sslmode=disable")
	err := run([]string{"ping"})
	if err == nil {
		t.Fatal("expected ping to fail on nonexistent database port")
	}
	if strings.Contains(err.Error(), "secret_password") {
		t.Fatal("ping error exposed credentials")
	}
}

func TestRunPingLive(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("TEST_MIGRATION_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run live ping test")
	}
	t.Setenv("DATABASE_URL", databaseURL)
	if err := run([]string{"ping"}); err != nil {
		t.Fatalf("live ping failed: %v", err)
	}
}
