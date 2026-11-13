package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type GoogleOIDC struct {
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
	client   *http.Client
}

func NewGoogleOIDC(ctx context.Context, clientID, clientSecret, baseURL string) (*GoogleOIDC, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	ctx = oidc.ClientContext(ctx, client)
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, errors.New("could not initialize Google OpenID Connect")
	}
	return &GoogleOIDC{config: oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: baseURL + "/api/auth/google/callback", Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}, verifier: provider.Verifier(&oidc.Config{ClientID: clientID, SupportedSigningAlgs: []string{"RS256"}}), client: client}, nil
}

func (g *GoogleOIDC) AuthURL(state, nonce, verifier string) string {
	return g.config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "select_account"))
}

func (g *GoogleOIDC) VerifyCode(ctx context.Context, code, verifier, nonceHash string) (GoogleIdentity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.client)
	token, err := g.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return GoogleIdentity{}, ErrGoogle
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return GoogleIdentity{}, ErrGoogle
	}
	id, err := g.verifier.Verify(ctx, raw)
	if err != nil {
		return GoogleIdentity{}, ErrGoogle
	}
	var claims struct {
		Email           string `json:"email"`
		EmailVerified   bool   `json:"email_verified"`
		Name            string `json:"name"`
		HostedDomain    string `json:"hd"`
		AuthorizedParty string `json:"azp"`
		IssuedAt        int64  `json:"iat"`
	}
	if err := id.Claims(&claims); err != nil {
		return GoogleIdentity{}, ErrGoogle
	}
	if subtle.ConstantTimeCompare([]byte(TokenHash(id.Nonce)), []byte(nonceHash)) != 1 || claims.IssuedAt <= 0 || claims.IssuedAt > time.Now().Add(time.Minute).Unix() || claims.IssuedAt < time.Now().Add(-15*time.Minute).Unix() || (claims.AuthorizedParty != "" && claims.AuthorizedParty != g.config.ClientID) || (len(id.Audience) > 1 && claims.AuthorizedParty == "") {
		return GoogleIdentity{}, ErrGoogle
	}
	if id.AccessTokenHash != "" {
		if err := id.VerifyAccessToken(token.AccessToken); err != nil {
			return GoogleIdentity{}, ErrGoogle
		}
	}
	identity := GoogleIdentity{Subject: id.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name, HostedDomain: claims.HostedDomain}
	if !validGoogleIdentity(identity) {
		return GoogleIdentity{}, ErrGoogle
	}
	return identity, nil
}
