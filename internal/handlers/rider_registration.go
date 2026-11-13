package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

type RiderRegistration struct {
	auth    *Auth
	service *services.RiderRegistration
}

func NewRiderRegistration(auth *Auth, service *services.RiderRegistration) *RiderRegistration {
	return &RiderRegistration{auth: auth, service: service}
}
func (h *RiderRegistration) Routes(mux *http.ServeMux) {
	protect := func(next http.HandlerFunc) http.Handler {
		return middleware.RequireAuth(h.auth.service, h.auth.cookieName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			next(w, r.WithContext(ctx))
		}))
	}
	mux.Handle("POST /api/rider/registration", protect(h.register))
	mux.Handle("GET /api/rider/registration", protect(h.own))
	mux.Handle("PATCH /api/rider/registration", protect(h.update))
}
func (h *RiderRegistration) fail(w http.ResponseWriter, err error) {
	var validation *services.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(w, 422, map[string]string{"error": validation.Error()})
	case errors.Is(err, services.ErrRiderDuplicate):
		writeJSON(w, 409, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrRiderProfileMissing):
		writeJSON(w, 404, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrRiderProfileLocked):
		writeJSON(w, 403, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, 503, map[string]string{"error": "Rider registration is temporarily unavailable."})
	}
}
func (h *RiderRegistration) register(w http.ResponseWriter, r *http.Request) {
	var input services.RiderInput
	if !decodeJSON(w, r, &input) {
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.service.Register(r.Context(), u.ID, input)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"rider": v, "email": u.Email})
}
func (h *RiderRegistration) own(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.service.Own(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"rider": v, "email": u.Email})
}
func (h *RiderRegistration) update(w http.ResponseWriter, r *http.Request) {
	var input services.RiderUpdate
	if !decodeJSON(w, r, &input) {
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.service.Update(r.Context(), u.ID, input)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"rider": v, "email": u.Email})
}
