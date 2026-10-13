package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"halisi/internal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	googleOAuthConfig  *oauth2.Config
	googleOIDCVerifier *oidc.IDTokenVerifier
)

func ConfigureGoogleAuth() error {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL"))

	if clientID == "" || clientSecret == "" {
		return errors.New("Google OAuth credentials are not configured")
	}

	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/google/callback"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return err
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID,
			"email",
			"profile",
		},
	}

	googleOIDCVerifier = provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return nil
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if googleOAuthConfig == nil {
		http.Error(w, "Google sign-in is unavailable", http.StatusServiceUnavailable)
		return
	}

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "Could not start Google sign-in", http.StatusInternalServerError)
		return
	}

	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	secure := strings.HasPrefix(googleOAuthConfig.RedirectURL, "https://")

	http.SetCookie(w, &http.Cookie{
		Name:     "google_oauth_state",
		Value:    state,
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	authURL := googleOAuthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if googleOAuthConfig == nil || googleOIDCVerifier == nil {
		http.Error(w, "Google sign-in is unavailable", http.StatusServiceUnavailable)
		return
	}

	stateCookie, err := r.Cookie("google_oauth_state")
	if err != nil || stateCookie.Value == "" ||
		r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	secure := strings.HasPrefix(googleOAuthConfig.RedirectURL, "https://")
	http.SetCookie(w, &http.Cookie{
		Name:     "google_oauth_state",
		Value:    "",
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	if r.URL.Query().Get("error") != "" {
		http.Redirect(
			w,
			r,
			"/login?error=google_signin_cancelled",
			http.StatusSeeOther,
		)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Google did not return an authorization code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	token, err := googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("Google token exchange failed: %v", err)
		http.Error(w, "Google sign-in failed", http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "Google did not return an ID token", http.StatusUnauthorized)
		return
	}

	idToken, err := googleOIDCVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("Google ID token verification failed: %v", err)
		http.Error(w, "Google identity could not be verified", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}

	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Could not read Google account information", http.StatusUnauthorized)
		return
	}

	if claims.Subject == "" || claims.Email == "" || !claims.EmailVerified {
		http.Error(w, "A verified Google email address is required", http.StatusForbidden)
		return
	}

	user, err := h.UserService.RegisterGoogleUser(
		claims.Name,
		claims.Email,
		claims.Subject,
	)
	if err != nil {
		log.Printf("Google account handling failed: %v", err)

		if strings.Contains(err.Error(), "already exists") {
			http.Redirect(w, r, "/login?error=account_exists", http.StatusSeeOther)
			return
		}

		http.Error(w, "Could not sign in with Google", http.StatusInternalServerError)
		return
	}

	if user.ID == 0 || !user.EmailVerified {
		http.Error(w, "Could not verify your account", http.StatusForbidden)
		return
	}

	session, err := h.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(w, "Could not create a session", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = user.ID
	session.Values["role"] = user.Role

	if err := session.Save(r, w); err != nil {
		http.Error(w, "Could not save your session", http.StatusInternalServerError)
		return
	}

	redirectByRole(w, r, user)
}

func redirectByRole(w http.ResponseWriter, r *http.Request, user models.User) {
	switch user.Role {
	case "admin":
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	case "owner":
		http.Redirect(w, r, "/shop/orders", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}
