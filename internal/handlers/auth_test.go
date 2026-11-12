package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

type fakeAuth struct{ calls int }

func (f *fakeAuth) Register(context.Context, services.RegisterInput) (services.User, error) {
	f.calls++
	return services.User{}, nil
}
func (f *fakeAuth) Login(context.Context, string, string, string) (services.LoginSession, error) {
	f.calls++
	return services.LoginSession{}, services.ErrInvalidCredentials
}
func (f *fakeAuth) Logout(context.Context, string) error { f.calls++; return nil }
func (f *fakeAuth) Authenticate(context.Context, string) (services.User, error) {
	return services.User{}, services.ErrUnauthenticated
}

func TestAuthRateLimits(t *testing.T) {
	fake := &fakeAuth{}
	a := handlers.NewAuth(fake, "test")
	handler := routes.New(nil, a.Routes)
	for i := 0; i < 6; i++ {
		r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"same@example.com","password":"wrong"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := http.StatusUnauthorized
		if i == 5 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("attempt %d returned %d, want %d", i, w.Code, want)
		}
		if i == 5 && w.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After")
		}
	}
	if fake.calls != 5 {
		t.Fatal("rate-limited attempt reached password verification")
	}
}

func TestAuthRequestSafety(t *testing.T) {
	for _, tt := range []struct {
		name, body, contentType, origin string
		want                            int
	}{
		{"cross-origin", `{}`, "application/json", "https://attacker.example", http.StatusForbidden},
		{"form submission", "email=x", "application/x-www-form-urlencoded", "", http.StatusUnsupportedMediaType},
		{"unknown role", `{"role":"admin"}`, "application/json", "", http.StatusBadRequest},
		{"multiple values", `{} {}`, "application/json", "", http.StatusBadRequest},
		{"oversized", `{"name":"` + strings.Repeat("a", 5000) + `"}`, "application/json", "", http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAuth{}
			a := handlers.NewAuth(fake, "test")
			r := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			routes.New(nil, a.Routes).ServeHTTP(w, r)
			if w.Code != tt.want || fake.calls != 0 {
				t.Fatalf("unsafe request returned %d and %d service calls", w.Code, fake.calls)
			}
		})
	}
}

func TestAuthPagesAndProtection(t *testing.T) {
	a := handlers.NewAuth(&fakeAuth{}, "test")
	handler := routes.New(nil, a.Routes)
	for _, path := range []string{"/login", "/register", "/static/css/auth.css", "/static/js/auth.js"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Body.Len() == 0 || w.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("page/asset unavailable: %s", path)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/auth/me", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatal("protected route accepted missing session")
	}
	w = httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/account", nil)
	r.Header.Set("Accept", "text/html")
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
		t.Fatal("unauthenticated page request was not redirected to login")
	}
}

func TestLoginIPRateLimit(t *testing.T) {
	fake := &fakeAuth{}
	a := handlers.NewAuth(fake, "test")
	handler := routes.New(nil, a.Routes)
	for i := 0; i < 11; i++ {
		body := fmt.Sprintf(`{"email":"user%d@example.com","password":"wrong"}`, i)
		r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := http.StatusUnauthorized
		if i == 10 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("attempt %d got %d, want %d", i, w.Code, want)
		}
	}
	if fake.calls != 10 {
		t.Fatal("IP rate-limited request reached password verification")
	}
}
