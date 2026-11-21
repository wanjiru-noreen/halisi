package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// CheckResult represents the outcome of an individual launch verification probe.
type CheckResult struct {
	Name     string        `json:"name"`
	Endpoint string        `json:"endpoint"`
	Passed   bool          `json:"passed"`
	Latency  time.Duration `json:"latency"`
	Details  string        `json:"details"`
}

// Config holds options for the launch verification tool.
type Config struct {
	BaseURL string
	Timeout time.Duration
	SkipDB  bool
}

func main() {
	defaultURL := os.Getenv("PUBLIC_BASE_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}

	urlFlag := flag.String("url", defaultURL, "Target base URL to verify (e.g. https://halisi.onrender.com or http://localhost:8080)")
	timeoutFlag := flag.Duration("timeout", 10*time.Second, "Timeout for each verification probe")
	skipDBFlag := flag.Bool("skip-db", false, "Skip the database health probe check")
	flag.Parse()

	cfg := Config{
		BaseURL: *urlFlag,
		Timeout: *timeoutFlag,
		SkipDB:  *skipDBFlag,
	}

	results, err := Run(context.Background(), http.DefaultClient, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Launch verification aborted: %v\n", err)
		os.Exit(1)
	}

	failed := false
	fmt.Printf("\n=== HALISI PRODUCTION LAUNCH VERIFICATION ===\n")
	fmt.Printf("Target: %s\n", cfg.BaseURL)
	fmt.Printf("Timestamp: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("%-32s %-8s %-12s %s\n", "CHECK", "STATUS", "LATENCY", "DETAILS")
	fmt.Printf("%s\n", strings.Repeat("-", 75))

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("%-32s %-8s %-12s %s\n", r.Name, status, r.Latency.Truncate(time.Microsecond), r.Details)
	}

	fmt.Printf("%s\n", strings.Repeat("-", 75))
	if failed {
		fmt.Printf("Result: FAILED - One or more launch verification checks did not succeed.\n\n")
		os.Exit(1)
	}
	fmt.Printf("Result: PASSED - All launch verification checks succeeded.\n\n")
}

// MaxAcceptableLatency defines the acceptable latency budget for health and storefront probes.
const MaxAcceptableLatency = 500 * time.Millisecond

// Run executes the complete launch verification check suite against target base URL.
func Run(ctx context.Context, client *http.Client, cfg Config) ([]CheckResult, error) {
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid base URL %q: must include scheme (http:// or https://) and host", cfg.BaseURL)
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	probeTimeout := cfg.Timeout
	if probeTimeout <= 0 {
		probeTimeout = 10 * time.Second
	}

	var results []CheckResult

	// 1. Liveness Probe: GET /health
	healthCtx, cancelHealth := context.WithTimeout(ctx, probeTimeout)
	resHealth := probeHealth(healthCtx, client, baseURL+"/health", "Liveness Probe (/health)")
	cancelHealth()
	results = append(results, resHealth)

	// 2. Database Probe: GET /health/db
	if !cfg.SkipDB {
		dbCtx, cancelDB := context.WithTimeout(ctx, probeTimeout)
		resDB := probeHealth(dbCtx, client, baseURL+"/health/db", "Database Probe (/health/db)")
		cancelDB()
		results = append(results, resDB)
	}

	// 3. Storefront Root: GET /
	rootCtx, cancelRoot := context.WithTimeout(ctx, probeTimeout)
	resRoot, rootHeaders := probeRoot(rootCtx, client, baseURL+"/")
	cancelRoot()
	results = append(results, resRoot)

	// 4. Security Headers Verification
	results = append(results, verifySecurityHeaders(rootHeaders, parsedURL.Scheme == "https")...)

	return results, nil
}

func probeHealth(ctx context.Context, client *http.Client, endpoint string, name string) CheckResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CheckResult{Name: name, Endpoint: endpoint, Passed: false, Latency: time.Since(start), Details: err.Error()}
	}

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{Name: name, Endpoint: endpoint, Passed: false, Latency: time.Since(start), Details: fmt.Sprintf("request error: %v", err)}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	latency := time.Since(start)
	if err != nil {
		return CheckResult{Name: name, Endpoint: endpoint, Passed: false, Latency: latency, Details: fmt.Sprintf("failed to read response body: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return CheckResult{
			Name:     name,
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("expected HTTP 200, got HTTP %d (%s)", resp.StatusCode, strings.TrimSpace(string(bodyBytes))),
		}
	}

	var statusResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(bodyBytes, &statusResp); err != nil || statusResp.Status != "ok" {
		return CheckResult{
			Name:     name,
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("invalid JSON body: expected {\"status\":\"ok\"}, got %q", string(bodyBytes)),
		}
	}

	if latency > MaxAcceptableLatency {
		return CheckResult{
			Name:     name,
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("latency %s exceeded %s budget", latency.Truncate(time.Millisecond), MaxAcceptableLatency),
		}
	}

	return CheckResult{
		Name:     name,
		Endpoint: endpoint,
		Passed:   true,
		Latency:  latency,
		Details:  "HTTP 200 - {\"status\":\"ok\"}",
	}
}

func probeRoot(ctx context.Context, client *http.Client, endpoint string) (CheckResult, http.Header) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CheckResult{Name: "Storefront Probe (/)", Endpoint: endpoint, Passed: false, Latency: time.Since(start), Details: err.Error()}, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{Name: "Storefront Probe (/)", Endpoint: endpoint, Passed: false, Latency: time.Since(start), Details: fmt.Sprintf("request error: %v", err)}, nil
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	latency := time.Since(start)
	if err != nil {
		return CheckResult{Name: "Storefront Probe (/)", Endpoint: endpoint, Passed: false, Latency: latency, Details: fmt.Sprintf("failed to read response body: %v", err)}, resp.Header
	}

	if resp.StatusCode != http.StatusOK {
		return CheckResult{
			Name:     "Storefront Probe (/)",
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("expected HTTP 200, got HTTP %d", resp.StatusCode),
		}, resp.Header
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		return CheckResult{
			Name:     "Storefront Probe (/)",
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("expected text/html Content-Type, got %q", contentType),
		}, resp.Header
	}

	if latency > MaxAcceptableLatency {
		return CheckResult{
			Name:     "Storefront Probe (/)",
			Endpoint: endpoint,
			Passed:   false,
			Latency:  latency,
			Details:  fmt.Sprintf("latency %s exceeded %s budget", latency.Truncate(time.Millisecond), MaxAcceptableLatency),
		}, resp.Header
	}

	return CheckResult{
		Name:     "Storefront Probe (/)",
		Endpoint: endpoint,
		Passed:   true,
		Latency:  latency,
		Details:  fmt.Sprintf("HTTP 200 - Content-Type: %s", contentType),
	}, resp.Header
}

func verifySecurityHeaders(headers http.Header, isHTTPS bool) []CheckResult {
	if headers == nil {
		return []CheckResult{
			{Name: "Security Headers Check", Passed: false, Details: "no response headers received"},
		}
	}

	var checks []CheckResult

	// X-Content-Type-Options: nosniff
	nosniff := headers.Get("X-Content-Type-Options") == "nosniff"
	checks = append(checks, CheckResult{
		Name:    "X-Content-Type-Options",
		Passed:  nosniff,
		Details: fmt.Sprintf("expected 'nosniff', got %q", headers.Get("X-Content-Type-Options")),
	})

	// X-Frame-Options: DENY
	xframe := headers.Get("X-Frame-Options") == "DENY"
	checks = append(checks, CheckResult{
		Name:    "X-Frame-Options",
		Passed:  xframe,
		Details: fmt.Sprintf("expected 'DENY', got %q", headers.Get("X-Frame-Options")),
	})

	// Referrer-Policy: same-origin
	referrer := headers.Get("Referrer-Policy") == "same-origin"
	checks = append(checks, CheckResult{
		Name:    "Referrer-Policy",
		Passed:  referrer,
		Details: fmt.Sprintf("expected 'same-origin', got %q", headers.Get("Referrer-Policy")),
	})

	// Content-Security-Policy
	csp := headers.Get("Content-Security-Policy") != ""
	checks = append(checks, CheckResult{
		Name:    "Content-Security-Policy",
		Passed:  csp,
		Details: fmt.Sprintf("present: %v", csp),
	})

	// Strict-Transport-Security (enforced on HTTPS)
	if isHTTPS {
		hsts := headers.Get("Strict-Transport-Security")
		passedHSTS := strings.Contains(hsts, "max-age=")
		checks = append(checks, CheckResult{
			Name:    "Strict-Transport-Security",
			Passed:  passedHSTS,
			Details: fmt.Sprintf("header: %q", hsts),
		})
	}

	return checks
}
