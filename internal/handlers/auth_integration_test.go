package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthenticationPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for the authentication integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not connect integration test pool")
	}
	defer admin.Close()
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("could not clean up auth test schema")
		}
	}()
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid test database URL")
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	m, err := database.NewMigrator(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatal("could not apply test migrations")
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal("could not connect auth test pool")
	}
	defer pool.Close()
	mailer := &captureMailer{}
	google := &fakeGoogle{flows: make(map[string]string)}
	service, err := services.NewAuth(pool, services.WithVerification(mailer, "https://halisi.example"), services.WithGoogle(google))
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(service, "production")
	server := httptest.NewTLSServer(routes.New(pool, auth.Routes))
	defer server.Close()
	client := server.Client()
	jar, _ := cookiejar.New(nil)
	client.Jar = jar
	const password = "a secure test passphrase"
	registration := map[string]any{"name": " Amina Wanjiru ", "email": " AMINA@example.com ", "phone": "+254712345678", "password": password}
	do := func(client *http.Client, method, path string, body any, cookie *http.Cookie) (*http.Response, []byte) {
		t.Helper()
		var reader io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			reader = bytes.NewReader(encoded)
		}
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		if body != nil {
			r.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), password) || strings.Contains(string(data), "password_hash") || strings.Contains(string(data), "token_hash") {
			t.Fatal("response exposed authentication secrets")
		}
		return response, data
	}
	response, _ := do(client, "POST", "/api/auth/register", registration, nil)
	if response.StatusCode != 201 {
		t.Fatalf("registration returned %d", response.StatusCode)
	}
	firstToken := mailer.lastToken(t)
	if _, err := pool.Exec(ctx, "UPDATE email_verification_tokens SET created_at=NOW()-INTERVAL '2 hours',expires_at=NOW()-INTERVAL '1 hour' WHERE token_hash=$1", services.TokenHash(firstToken)); err != nil {
		t.Fatal(err)
	}
	response, _ = do(client, "POST", "/api/auth/verify-email", map[string]string{"token": firstToken}, nil)
	if response.StatusCode != 400 {
		t.Fatal("expired verification token accepted")
	}
	response, _ = do(client, "POST", "/api/auth/verification/resend", map[string]string{"email": "amina@example.com"}, nil)
	if response.StatusCode != 202 {
		t.Fatal("resend failed")
	}
	secondToken := mailer.lastToken(t)
	response, _ = do(client, "POST", "/api/auth/verification/resend", map[string]string{"email": "amina@example.com"}, nil)
	if response.StatusCode != 202 {
		t.Fatal("second resend failed")
	}
	response, _ = do(client, "POST", "/api/auth/verify-email", map[string]string{"token": secondToken}, nil)
	if response.StatusCode != 400 {
		t.Fatal("superseded verification token accepted")
	}
	verificationToken := mailer.lastToken(t)
	response, _ = do(client, "POST", "/api/auth/verify-email", map[string]string{"token": verificationToken}, nil)
	if response.StatusCode != 200 {
		t.Fatal("email verification failed")
	}
	response, _ = do(client, "POST", "/api/auth/verify-email", map[string]string{"token": verificationToken}, nil)
	if response.StatusCode != 400 {
		t.Fatal("reused verification token accepted")
	}
	var verified bool
	if err := pool.QueryRow(ctx, "SELECT email_verified_at IS NOT NULL FROM users WHERE email=$1", "amina@example.com").Scan(&verified); err != nil || !verified {
		t.Fatal("verification not persisted")
	}
	var hash, role, phone string
	if err := pool.QueryRow(ctx, "SELECT password_hash,role,phone FROM users WHERE email=$1", "amina@example.com").Scan(&hash, &role, &phone); err != nil {
		t.Fatal(err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != services.PasswordCost || hash == password || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || role != "customer" || phone != "+254712345678" {
		t.Fatal("password hashing or persisted registration data is incorrect")
	}
	response, _ = do(client, "POST", "/api/auth/register", registration, nil)
	if response.StatusCode != 409 {
		t.Fatal("duplicate registration accepted")
	}
	invalid := map[string]string{"name": "Amina", "email": "invalid", "phone": "+254712345678", "password": password}
	response, _ = do(client, "POST", "/api/auth/register", invalid, nil)
	if response.StatusCode != 422 {
		t.Fatal("invalid registration accepted")
	}
	response, wrongBody := do(client, "POST", "/api/auth/login", map[string]string{"email": "amina@example.com", "password": "wrong-password"}, nil)
	if response.StatusCode != 401 {
		t.Fatal("wrong password accepted")
	}
	response, missingBody := do(client, "POST", "/api/auth/login", map[string]string{"email": "missing@example.com", "password": "wrong-password"}, nil)
	if response.StatusCode != 401 || !bytes.Equal(wrongBody, missingBody) {
		t.Fatal("missing account returned a different credential error")
	}
	login := map[string]string{"email": "AMINA@example.com", "password": password}
	response, _ = do(client, "POST", "/api/auth/login", login, nil)
	if response.StatusCode != 200 || len(response.Cookies()) != 1 {
		t.Fatal("login failed")
	}
	first := response.Cookies()[0]
	if first.Name != "__Host-halisi_session" || !first.Secure || !first.HttpOnly || first.SameSite != http.SameSiteLaxMode || first.Path != "/" || first.Domain != "" || first.MaxAge < 86000 || !services.ValidToken(first.Value) {
		t.Fatal("session cookie security attributes are incorrect")
	}
	var storedHash string
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE token_hash=$1", services.TokenHash(first.Value)).Scan(&storedHash); err != nil || storedHash == first.Value {
		t.Fatal("session token was not stored as a hash")
	}
	response, body := do(client, "GET", "/api/auth/me", nil, nil)
	if response.StatusCode != 200 || !strings.Contains(string(body), "amina@example.com") || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("cookie session was not persisted across requests")
	}
	restarted, err := services.NewAuth(pool)
	if err != nil {
		t.Fatal(err)
	}
	if user, err := restarted.Authenticate(ctx, first.Value); err != nil || user.Email != "amina@example.com" {
		t.Fatal("session did not survive authentication service restart")
	}
	response, _ = do(client, "POST", "/api/auth/login", login, nil)
	if response.StatusCode != 200 {
		t.Fatal("second login failed")
	}
	second := response.Cookies()[0]
	if second.Value == first.Value {
		t.Fatal("login reused a session token")
	}
	raw := &http.Client{Transport: client.Transport}
	response, _ = do(raw, "GET", "/api/auth/me", nil, first)
	if response.StatusCode != 401 {
		t.Fatal("rotated session remained active")
	}
	response, _ = do(client, "POST", "/api/auth/logout", map[string]string{}, nil)
	if response.StatusCode != 204 || response.Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear cookie")
	}
	response, _ = do(raw, "GET", "/api/auth/me", nil, second)
	if response.StatusCode != 401 {
		t.Fatal("logged-out session remained active")
	}
	response, _ = do(client, "GET", "/api/auth/me", nil, nil)
	if response.StatusCode != 401 {
		t.Fatal("missing cookie passed protected route")
	}
	response, _ = do(client, "POST", "/api/auth/login", login, nil)
	if response.StatusCode != 200 {
		t.Fatal("expiry-test login failed")
	}
	third := response.Cookies()[0]
	if _, err := pool.Exec(ctx, "UPDATE sessions SET created_at=NOW()-INTERVAL '2 days',expires_at=NOW()-INTERVAL '1 day' WHERE token_hash=$1", services.TokenHash(third.Value)); err != nil {
		t.Fatal(err)
	}
	response, _ = do(client, "GET", "/api/auth/me", nil, nil)
	if response.StatusCode != 401 {
		t.Fatal("expired session accepted")
	}
	response, _ = do(raw, "GET", "/api/auth/me", nil, &http.Cookie{Name: first.Name, Value: "forged-token"})
	if response.StatusCode != 401 {
		t.Fatal("forged token accepted")
	}
	runGoogleChecks(t, ctx, pool, service, mailer)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, body = do(client, "POST", "/api/auth/google/start", map[string]string{}, nil)
	if response.StatusCode != 200 || len(response.Cookies()) != 1 || response.Cookies()[0].Name != "__Host-halisi_oauth" || !response.Cookies()[0].HttpOnly || !response.Cookies()[0].Secure {
		t.Fatal("OAuth browser binding cookie was not set securely")
	}
	var flowURL struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &flowURL); err != nil {
		t.Fatal(err)
	}
	providerURL, err := url.Parse(flowURL.URL)
	if err != nil {
		t.Fatal(err)
	}
	callback := "/api/auth/google/callback?state=" + providerURL.Query().Get("state") + "&code=new"
	response, _ = do(client, "GET", callback, nil, nil)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/account?google=connected" {
		t.Fatal("OAuth callback did not establish a browser session")
	}
	response, body = do(client, "GET", "/api/auth/me", nil, nil)
	if response.StatusCode != 200 || !strings.Contains(string(body), "new@gmail.com") {
		t.Fatal("Google browser session did not authenticate")
	}
	response, _ = do(client, "GET", callback, nil, nil)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/login?error=google" {
		t.Fatal("OAuth callback replay accepted")
	}
	mailer.fail = true
	registration["email"] = "delivery-failure@example.com"
	response, _ = do(client, "POST", "/api/auth/register", registration, nil)
	if response.StatusCode != 503 {
		t.Fatal("SMTP failure was not reported")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE email=$1", "delivery-failure@example.com").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed delivery left a partially registered account")
	}
}
