package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"Content-Security-Policy":   "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "same-origin",
		"X-Frame-Options":           "DENY",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains; preload",
	}

	for header, expectedValue := range expectedHeaders {
		if got := rec.Header().Get(header); got != expectedValue {
			t.Errorf("expected header %q = %q, got %q", header, expectedValue, got)
		}
	}
}
