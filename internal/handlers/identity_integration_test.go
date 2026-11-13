package handlers_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

type captureMailer struct {
	links []string
	fail  bool
}

func (m *captureMailer) SendVerification(_ context.Context, _ string, link string) error {
	if m.fail {
		return errors.New("SMTP unavailable")
	}
	m.links = append(m.links, link)
	return nil
}
func (m *captureMailer) lastToken(t *testing.T) string {
	t.Helper()
	if len(m.links) == 0 {
		t.Fatal("verification email not sent")
	}
	u, err := url.Parse(m.links[len(m.links)-1])
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "halisi.example" || u.RawQuery != "" {
		t.Fatal("verification link did not use trusted origin/fragment")
	}
	return strings.TrimPrefix(u.Fragment, "token=")
}

type fakeGoogle struct{ flows map[string]string }

func (g *fakeGoogle) AuthURL(state, nonce, verifier string) string {
	g.flows[verifier] = services.TokenHash(nonce)
	return "https://accounts.google.com/auth?state=" + state
}
func (g *fakeGoogle) VerifyCode(_ context.Context, code, verifier, nonceHash string) (services.GoogleIdentity, error) {
	if g.flows[verifier] != nonceHash {
		return services.GoogleIdentity{}, services.ErrGoogle
	}
	i := services.GoogleIdentity{Subject: code, Email: "new@gmail.com", EmailVerified: true, Name: "Google customer"}
	switch code {
	case "existing", "hijack":
		i.Email = "amina@example.com"
	case "changed":
		i.Subject = "existing"
		i.Email = "changed@gmail.com"
	case "third":
		i.Email = "third@example.invalid"
	case "unverified":
		i.EmailVerified = false
	}
	return i, nil
}

func runGoogleChecks(t *testing.T, ctx context.Context, pool *pgxpool.Pool, a *services.Auth, mailer *captureMailer) {
	t.Helper()
	finish := func(flow services.GoogleFlow, code, session string) (services.LoginSession, bool, error) {
		u, err := url.Parse(flow.URL)
		if err != nil {
			t.Fatal(err)
		}
		return a.FinishGoogle(ctx, u.Query().Get("state"), flow.BrowserToken, code, session)
	}
	flow, err := a.StartGoogle(ctx, "", false)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(flow.URL)
	if _, _, err := a.FinishGoogle(ctx, u.Query().Get("state"), strings.Repeat("A", 43), "new", ""); !errors.Is(err, services.ErrGoogle) {
		t.Fatal("wrong browser binding accepted")
	}
	first, _, err := finish(flow, "new", "")
	if err != nil || first.User.Email != "new@gmail.com" || !first.User.EmailVerified {
		t.Fatalf("new Google login failed: %v", err)
	}
	if _, _, err := finish(flow, "new", ""); !errors.Is(err, services.ErrGoogle) {
		t.Fatal("OAuth state replay accepted")
	}
	if _, err := a.Login(ctx, "new@gmail.com", "a secure test passphrase", ""); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Fatal("Google-only account accepted password login")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	u, _ = url.Parse(flow.URL)
	if _, err := pool.Exec(ctx, "UPDATE oauth_states SET created_at=NOW()-INTERVAL '20 minutes',expires_at=NOW()-INTERVAL '10 minutes' WHERE state_hash=$1", services.TokenHash(u.Query().Get("state"))); err != nil {
		t.Fatal(err)
	}
	if _, _, err := finish(flow, "new", ""); !errors.Is(err, services.ErrGoogle) {
		t.Fatal("expired OAuth state accepted")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	if _, _, err := finish(flow, "unverified", ""); !errors.Is(err, services.ErrGoogle) {
		t.Fatal("unverified Google email accepted")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	if _, _, err := finish(flow, "existing", ""); !errors.Is(err, services.ErrLinkRequired) {
		t.Fatal("Google auto-linked an existing email")
	}
	if _, err := a.StartGoogle(ctx, "", true); !errors.Is(err, services.ErrRecentLogin) {
		t.Fatal("anonymous account linking accepted")
	}
	local, err := a.Login(ctx, "amina@example.com", "a secure test passphrase", "")
	if err != nil {
		t.Fatal(err)
	}
	flow, err = a.StartGoogle(ctx, local.Token, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Logout(ctx, local.Token); err != nil {
		t.Fatal(err)
	}
	if _, _, err := finish(flow, "existing", local.Token); !errors.Is(err, services.ErrRecentLogin) {
		t.Fatal("link callback accepted revoked session")
	}
	local, err = a.Login(ctx, "amina@example.com", "a secure test passphrase", "")
	if err != nil {
		t.Fatal(err)
	}
	flow, err = a.StartGoogle(ctx, local.Token, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := finish(flow, "existing", first.Token); !errors.Is(err, services.ErrRecentLogin) {
		t.Fatal("link callback accepted different session")
	}
	flow, err = a.StartGoogle(ctx, local.Token, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, linked, err := finish(flow, "existing", local.Token); err != nil || !linked {
		t.Fatalf("explicit linking failed: %v", err)
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	signedIn, _, err := finish(flow, "existing", "")
	if err != nil || signedIn.User.ID != local.User.ID {
		t.Fatal("linked Google identity did not authenticate existing user")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	if _, _, err := finish(flow, "hijack", ""); !errors.Is(err, services.ErrLinkRequired) {
		t.Fatal("second Google subject took over account by email")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	changed, _, err := finish(flow, "changed", "")
	if err != nil || changed.User.ID != local.User.ID || changed.User.Email != "amina@example.com" {
		t.Fatal("Google email change altered stable-subject identity")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	if _, _, err := finish(flow, "third", ""); !errors.Is(err, services.ErrEmailVerificationRequired) {
		t.Fatal("third-party Google email was incorrectly treated as authoritative")
	}
	if err := a.VerifyEmail(ctx, mailer.lastToken(t)); err != nil {
		t.Fatal("third-party Google email could not be verified")
	}
	flow, _ = a.StartGoogle(ctx, "", false)
	third, _, err := finish(flow, "third", "")
	if err != nil || !third.User.EmailVerified {
		t.Fatal("verified third-party Google email could not sign in")
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET email='different@example.invalid' WHERE id=$1", third.User.ID); err != nil {
		t.Fatal(err)
	}
	var verified bool
	if err := pool.QueryRow(ctx, "SELECT email_verified_at IS NOT NULL FROM users WHERE id=$1", third.User.ID).Scan(&verified); err != nil || verified {
		t.Fatal("email change retained verification")
	}
	if _, err := pool.Exec(ctx, "UPDATE sessions SET created_at=NOW()-INTERVAL '11 minutes' WHERE token_hash=$1", services.TokenHash(local.Token)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartGoogle(ctx, local.Token, true); !errors.Is(err, services.ErrRecentLogin) {
		t.Fatal("stale authentication allowed account linking")
	}
}
