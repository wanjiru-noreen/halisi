package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/services"
)

type Authenticator interface {
	Authenticate(context.Context, string) (services.User, error)
}

type userKey struct{}

func CurrentUser(ctx context.Context) (services.User, bool) {
	u, ok := ctx.Value(userKey{}).(services.User)
	return u, ok
}

func RequireAuth(auth Authenticator, cookieName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			authRequired(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		user, err := auth.Authenticate(ctx, cookie.Value)
		if errors.Is(err, services.ErrUnauthenticated) {
			authRequired(w, r)
			return
		}
		if err != nil {
			authError(w, http.StatusServiceUnavailable, "Authentication is temporarily unavailable.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func authError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func authRequired(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/") && strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	authError(w, http.StatusUnauthorized, "Authentication required.")
}
