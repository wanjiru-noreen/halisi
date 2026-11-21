package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
)

func BenchmarkSecurityHeaders(b *testing.B) {
	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.SecurityHeaders(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkSecurityHeadersHTTPS(b *testing.B) {
	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.SecurityHeaders(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkAttemptLimiter(b *testing.B) {
	limiter := middleware.NewAttemptLimiter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = limiter.Allow("benchmark-key", 1000000000)
	}
}
