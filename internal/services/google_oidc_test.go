package services

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func TestGoogleIDTokenValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	nonce, _ := randomToken()
	verifier := oauth2.GenerateVerifier()
	var raw string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("code_verifier") != verifier || r.Form.Get("code") != "test-code" {
			http.Error(w, "invalid grant", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-access-token", "token_type": "Bearer", "id_token": raw})
	}))
	defer tokenServer.Close()
	g := &GoogleOIDC{config: oauth2.Config{ClientID: "test-client", ClientSecret: "test-secret", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/auth", TokenURL: tokenServer.URL, AuthStyle: oauth2.AuthStyleInParams}}, verifier: oidc.NewVerifier("https://accounts.google.com", &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "test-client", SupportedSigningAlgs: []string{"RS256"}}), client: tokenServer.Client()}
	authorization, _ := url.Parse(g.AuthURL("state", nonce, verifier))
	if authorization.Query().Get("nonce") != nonce || authorization.Query().Get("state") != "state" || authorization.Query().Get("code_challenge_method") != "S256" || authorization.Query().Get("code_challenge") != oauth2.S256ChallengeFromVerifier(verifier) {
		t.Fatal("missing OAuth protections")
	}
	for _, name := range []string{"valid", "issuer", "audience", "expiry", "nonce", "signature", "unverified email", "authorized party", "future issued", "multiple audiences", "access token hash"} {
		t.Run(name, func(t *testing.T) {
			claims := map[string]any{"iss": "https://accounts.google.com", "aud": "test-client", "sub": "google-subject", "email": "test@gmail.com", "email_verified": true, "nonce": nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()}
			signingKey := key
			switch name {
			case "issuer":
				claims["iss"] = "https://attacker.invalid"
			case "audience":
				claims["aud"] = "other-client"
			case "expiry":
				claims["exp"] = time.Now().Add(-time.Minute).Unix()
			case "nonce":
				claims["nonce"] = "wrong-nonce"
			case "signature":
				signingKey = other
			case "unverified email":
				claims["email_verified"] = false
			case "authorized party":
				claims["azp"] = "other-client"
			case "future issued":
				claims["iat"] = time.Now().Add(time.Hour).Unix()
			case "multiple audiences":
				claims["aud"] = []string{"test-client", "another"}
			case "access token hash":
				claims["at_hash"] = "invalid"
			}
			payload, _ := json.Marshal(claims)
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
			signed := header + "." + base64.RawURLEncoding.EncodeToString(payload)
			digest := sha256.Sum256([]byte(signed))
			signature, err := rsa.SignPKCS1v15(rand.Reader, signingKey, crypto.SHA256, digest[:])
			if err != nil {
				t.Fatal(err)
			}
			raw = signed + "." + base64.RawURLEncoding.EncodeToString(signature)
			identity, err := g.VerifyCode(context.Background(), "test-code", verifier, TokenHash(nonce))
			if name == "valid" {
				if err != nil || identity.Subject != "google-subject" {
					t.Fatalf("valid token rejected: %v", err)
				}
			} else if err == nil {
				t.Fatal("unsafe ID token accepted")
			}
		})
	}
}
