package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type AuthService interface {
	middleware.Authenticator
	Register(context.Context, services.RegisterInput) (services.User, error)
	Login(context.Context, string, string, string) (services.LoginSession, error)
	Logout(context.Context, string) error
}

type Auth struct {
	service    AuthService
	secure     bool
	cookieName string
	limiter    *middleware.AttemptLimiter
}

func NewAuth(service AuthService, appEnv string) *Auth {
	name := "halisi_session"
	if appEnv == "production" {
		name = "__Host-halisi_session"
	}
	return &Auth{service: service, secure: appEnv == "production", cookieName: name, limiter: middleware.NewAttemptLimiter()}
}

func (a *Auth) Routes(mux *http.ServeMux) {
	a.identityRoutes(mux)
	mux.HandleFunc("POST /api/auth/register", a.Register)
	mux.HandleFunc("POST /api/auth/login", a.Login)
	mux.HandleFunc("POST /api/auth/logout", a.Logout)
	mux.Handle("GET /api/auth/me", middleware.RequireAuth(a.service, a.cookieName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := middleware.CurrentUser(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"user": user})
	})))
	mux.Handle("GET /login", webassets.Page("login"))
	mux.Handle("GET /register", webassets.Page("register"))
	mux.Handle("GET /account", middleware.RequireAuth(a.service, a.cookieName, webassets.Page("account")))
	mux.Handle("GET /static/", webassets.Static())
}

func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	if !a.limiter.Allow("register:"+clientIP(r), 10) {
		rateLimited(w)
		return
	}
	var input services.RegisterInput
	if !decodeJSON(w, r, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := a.service.Register(ctx, input)
	if err != nil {
		a.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	if !a.limiter.Allow("login-ip:"+clientIP(r), 10) {
		rateLimited(w)
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !a.limiter.Allow("login-email:"+services.TokenHash(services.NormalizeEmail(input.Email)), 5) {
		rateLimited(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	session, err := a.service.Login(ctx, input.Email, input.Password, a.token(r))
	if err != nil {
		a.writeError(w, err)
		return
	}
	a.setSession(w, session)
	redirect, ok := models.DashboardPath(models.Role(session.User.Role))
	if !ok {
		redirect = "/dashboard"
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": session.User, "redirect": redirect})
}

func (a *Auth) setSession(w http.ResponseWriter, session services.LoginSession) {
	cookie := a.cookie(session.Token)
	cookie.Expires = session.ExpiresAt.UTC()
	cookie.MaxAge = int(time.Until(session.ExpiresAt).Seconds())
	if cookie.MaxAge < 1 {
		cookie.MaxAge = 1
	}
	http.SetCookie(w, cookie)
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	// Requiring JSON prevents cross-site form submissions even in older browsers.
	var input struct{}
	if !decodeJSON(w, r, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := a.service.Logout(ctx, a.token(r)); err != nil {
		a.writeError(w, err)
		return
	}
	cookie := a.cookie("")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0).UTC()
	http.SetCookie(w, cookie)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *Auth) token(r *http.Request) string {
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (a *Auth) cookie(value string) *http.Cookie {
	return &http.Cookie{Name: a.cookieName, Value: value, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode}
}

func (a *Auth) writeError(w http.ResponseWriter, err error) {
	var validation *services.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": validation.Error()})
	case errors.Is(err, services.ErrEmailExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid email or password."})
	case errors.Is(err, services.ErrInvalidVerification):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrRecentLogin):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrLinkRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrGoogle):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrEmailDelivery):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Authentication is temporarily unavailable. Please try again."})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json."})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(target)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = errors.New("unexpected JSON value")
		}
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON request. Only the documented fields are accepted (maximum 4 KiB)."})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "unknown"
	}
	return host
}

func rateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", strconv.Itoa(15*60))
	writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many attempts. Try again in 15 minutes."})
}
