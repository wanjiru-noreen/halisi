package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestDatabaseHealth(t *testing.T) {
	for _, available := range []bool{true, false} {
		ping := pingFunc(func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("database ping must have a deadline")
			}
			if !available {
				return errors.New("driver error containing secret credentials")
			}
			return nil
		})
		response := httptest.NewRecorder()
		New(ping).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/db", nil))
		wantStatus, wantBody := http.StatusOK, `{"status":"ok"}`
		if !available {
			wantStatus, wantBody = http.StatusServiceUnavailable, `{"status":"unavailable"}`
		}
		if response.Code != wantStatus || response.Body.String() != wantBody {
			t.Fatalf("database health returned %d %q", response.Code, response.Body.String())
		}
		if response.Header().Get("Content-Type") != "application/json" {
			t.Error("expected JSON response")
		}
	}
}

func TestHealth(t *testing.T) {
	response := httptest.NewRecorder()
	New(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := response.Body.String(); got != `{"status":"ok"}` {
		t.Errorf("body = %q, want health JSON", got)
	}
}
