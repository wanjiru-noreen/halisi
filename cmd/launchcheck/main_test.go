package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLaunchCheck_AllPassing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		switch r.URL.Path {
		case "/health", "/health/db":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>Halisi</body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
		SkipDB:  false,
	}

	results, err := Run(context.Background(), ts.Client(), cfg)
	if err != nil {
		t.Fatalf("unexpected error running checks: %v", err)
	}

	if len(results) < 6 {
		t.Fatalf("expected at least 6 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Passed {
			t.Errorf("expected check %q to pass, but failed: %s", r.Name, r.Details)
		}
	}
}

func TestLaunchCheck_HealthFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
		SkipDB:  true,
	}

	results, err := Run(context.Background(), ts.Client(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var healthCheck *CheckResult
	for i := range results {
		if results[i].Name == "Liveness Probe (/health)" {
			healthCheck = &results[i]
			break
		}
	}

	if healthCheck == nil {
		t.Fatal("health check result not found")
	}
	if healthCheck.Passed {
		t.Error("expected liveness check to fail on 503")
	}
}

func TestLaunchCheck_SkipDB(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
		SkipDB:  true,
	}

	results, err := Run(context.Background(), ts.Client(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range results {
		if r.Name == "Database Probe (/health/db)" {
			t.Error("database probe should have been skipped")
		}
	}
}

func TestLaunchCheck_MissingSecurityHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
		SkipDB:  true,
	}

	results, err := Run(context.Background(), ts.Client(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	failedHeaders := 0
	for _, r := range results {
		if !r.Passed && (r.Name == "X-Content-Type-Options" || r.Name == "X-Frame-Options" || r.Name == "Referrer-Policy") {
			failedHeaders++
		}
	}

	if failedHeaders != 3 {
		t.Errorf("expected 3 security header failures, got %d", failedHeaders)
	}
}

func TestLaunchCheck_HTTPS_HSTS(t *testing.T) {
	headers := make(http.Header)
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
	headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	headers.Set("Content-Security-Policy", "default-src 'self'")
	headers.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

	checks := verifySecurityHeaders(headers, true)
	for _, c := range checks {
		if !c.Passed {
			t.Errorf("check %q failed unexpectedly: %s", c.Name, c.Details)
		}
	}

	// Test missing HSTS on HTTPS
	headers.Del("Strict-Transport-Security")
	checksMissing := verifySecurityHeaders(headers, true)
	var hstsCheck *CheckResult
	for i := range checksMissing {
		if checksMissing[i].Name == "Strict-Transport-Security" {
			hstsCheck = &checksMissing[i]
			break
		}
	}
	if hstsCheck == nil || hstsCheck.Passed {
		t.Error("expected HSTS check to fail when header is missing on HTTPS")
	}
}

func TestLaunchCheck_NonHTMLStorefront(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		if r.URL.Path == "/" {
			// Returns JSON instead of HTML
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"not html"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
		SkipDB:  true,
	}

	results, err := Run(context.Background(), ts.Client(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var storefrontCheck *CheckResult
	for i := range results {
		if results[i].Name == "Storefront Probe (/)" {
			storefrontCheck = &results[i]
			break
		}
	}

	if storefrontCheck == nil {
		t.Fatal("storefront check result not found")
	}
	if storefrontCheck.Passed {
		t.Error("expected storefront check to fail when Content-Type is not text/html")
	}
}

func TestLaunchCheck_InvalidURL(t *testing.T) {
	cfg := Config{
		BaseURL: "not-a-valid-url",
		Timeout: 5 * time.Second,
	}
	_, err := Run(context.Background(), http.DefaultClient, cfg)
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
