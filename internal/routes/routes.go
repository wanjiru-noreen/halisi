package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type Pinger interface {
	Ping(context.Context) error
}

// New creates the server's HTTP routes.
func New(database Pinger, register ...func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", webassets.Page("pages/home"))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /health/db", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if database == nil || database.Ping(ctx) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	for _, routes := range register {
		routes(mux)
	}
	return middleware.SecurityHeaders(http.NewCrossOriginProtection().Handler(mux))
}
