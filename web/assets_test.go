package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webassets "github.com/wanjiru-noreen/halisi/web"
)

func TestPageRendering(t *testing.T) {
	tests := []struct {
		name       string
		page       string
		wantCode   int
		wantSubstr []string
	}{
		{
			name:     "rider dashboard",
			page:     "rider/dashboard",
			wantCode: http.StatusOK,
			wantSubstr: []string{
				"Rider Dashboard",
				"Active Deliveries",
				"data-page=\"rider-dashboard\"",
				"/static/js/rider-dashboard.js",
			},
		},
		{
			name:     "rider deliveries",
			page:     "rider/deliveries",
			wantCode: http.StatusOK,
			wantSubstr: []string{
				"Assigned Deliveries",
				"data-page=\"rider-deliveries\"",
				"/static/js/rider-deliveries.js",
			},
		},
		{
			name:     "customer dashboard",
			page:     "customer/dashboard",
			wantCode: http.StatusOK,
			wantSubstr: []string{
				"Customer dashboard",
				"data-page=\"customer-dashboard\"",
			},
		},
		{
			name:     "auth login",
			page:     "login",
			wantCode: http.StatusOK,
			wantSubstr: []string{
				"Log in",
			},
		},
		{
			name:     "nonexistent page",
			page:     "nonexistent/page",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "path traversal attempt",
			page:     "../etc/passwd",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := webassets.Page(tt.page)
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("page %q returned %d, want %d", tt.page, rec.Code, tt.wantCode)
			}

			if tt.wantCode == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if !strings.HasPrefix(ct, "text/html") {
					t.Fatalf("expected text/html Content-Type, got %q", ct)
				}
				body := rec.Body.String()
				for _, sub := range tt.wantSubstr {
					if !strings.Contains(body, sub) {
						t.Fatalf("expected page %q body to contain %q", tt.page, sub)
					}
				}
			}
		})
	}
}
