package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/services"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type identityService interface {
	VerificationEnabled() bool
	GoogleEnabled() bool
	ResendVerification(context.Context, string) error
	VerifyEmail(context.Context, string) error
	StartGoogle(context.Context, string, bool) (services.GoogleFlow, error)
	FinishGoogle(context.Context, string, string, string, string) (services.LoginSession, bool, error)
}

func (a *Auth) identityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/options", func(w http.ResponseWriter, r *http.Request) {
		identity, ok := a.service.(identityService)
		writeJSON(w, 200, map[string]bool{"google_enabled": ok && identity.GoogleEnabled(), "verification_enabled": ok && identity.VerificationEnabled()})
	})
	mux.HandleFunc("POST /api/auth/verification/resend", a.resendVerification)
	mux.HandleFunc("POST /api/auth/verify-email", a.verifyEmail)
	mux.HandleFunc("POST /api/auth/google/start", func(w http.ResponseWriter, r *http.Request) { a.startGoogle(w, r, false) })
	mux.HandleFunc("POST /api/auth/google/link", func(w http.ResponseWriter, r *http.Request) { a.startGoogle(w, r, true) })
	mux.HandleFunc("GET /api/auth/google/callback", a.googleCallback)
	mux.Handle("GET /verify-email", webassets.Page("verify"))
}

func (a *Auth) resendVerification(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !a.limiter.Allow("verification-ip:"+clientIP(r), 5) || !a.limiter.Allow("verification-email:"+services.TokenHash(services.NormalizeEmail(input.Email)), 3) {
		rateLimited(w)
		return
	}
	identity, ok := a.service.(identityService)
	if !ok {
		a.writeError(w, services.ErrEmailDelivery)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := identity.ResendVerification(ctx, input.Email); err != nil {
		a.writeError(w, err)
		return
	}
	writeJSON(w, 202, map[string]string{"message": "If this account needs verification, a new email has been sent."})
}

func (a *Auth) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !a.limiter.Allow("verify:"+clientIP(r), 20) {
		rateLimited(w)
		return
	}
	identity, ok := a.service.(identityService)
	if !ok {
		a.writeError(w, services.ErrInvalidVerification)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := identity.VerifyEmail(ctx, input.Token); err != nil {
		a.writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Your email address is verified."})
}

func (a *Auth) oauthCookie(value string) *http.Cookie {
	cookie := a.cookie(value)
	cookie.Name = "halisi_oauth"
	if a.secure {
		cookie.Name = "__Host-halisi_oauth"
	}
	cookie.MaxAge = 600
	cookie.Expires = time.Now().UTC().Add(10 * time.Minute)
	return cookie
}

func (a *Auth) startGoogle(w http.ResponseWriter, r *http.Request, link bool) {
	var input struct{}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !a.limiter.Allow("google:"+clientIP(r), 10) {
		rateLimited(w)
		return
	}
	identity, ok := a.service.(identityService)
	if !ok {
		a.writeError(w, services.ErrGoogle)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	flow, err := identity.StartGoogle(ctx, a.token(r), link)
	if err != nil {
		a.writeError(w, err)
		return
	}
	http.SetCookie(w, a.oauthCookie(flow.BrowserToken))
	writeJSON(w, 200, map[string]string{"url": flow.URL})
}

func (a *Auth) googleCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	cookie := a.oauthCookie("")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0)
	http.SetCookie(w, cookie)
	binding, err := r.Cookie(cookie.Name)
	identity, ok := a.service.(identityService)
	if err != nil || !ok {
		http.Redirect(w, r, "/login?error=google", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	code := r.URL.Query().Get("code")
	if r.URL.Query().Get("error") != "" {
		code = ""
	}
	session, linked, err := identity.FinishGoogle(ctx, r.URL.Query().Get("state"), binding.Value, code, a.token(r))
	if err != nil {
		reason := "google"
		if errors.Is(err, services.ErrLinkRequired) {
			reason = "link_required"
		}
		if errors.Is(err, services.ErrRecentLogin) {
			reason = "reauth"
		}
		http.Redirect(w, r, "/login?error="+reason, http.StatusSeeOther)
		return
	}
	if !linked {
		a.setSession(w, session)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/account?google=connected", http.StatusSeeOther)
}
