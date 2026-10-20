package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleAuth_NotConfigured(t *testing.T) {
	ResetGoogleAuthForTesting()

	handler := &Handler{}

	t.Run("GoogleLogin_Returns503WhenNotConfigured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
		rr := httptest.NewRecorder()

		handler.GoogleLogin(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}
	})

	t.Run("GoogleCallback_Returns503WhenNotConfigured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=mock-code&state=mock-state", nil)
		rr := httptest.NewRecorder()

		handler.GoogleCallback(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}
	})
}
