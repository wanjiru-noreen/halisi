package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDarajaTimestampUsesNairobiTime(t *testing.T) {
	utc := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got, want := darajaTimestamp(utc), "20260101030000"; got != want {
		t.Fatalf("Daraja timestamp = %q, want Nairobi time %q", got, want)
	}
}

func TestDarajaInitiate(t *testing.T) {
	var sawSTK bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			if r.Method != http.MethodGet {
				t.Errorf("OAuth method = %s, want GET", r.Method)
			}
			user, password, ok := r.BasicAuth()
			if !ok || user != "consumer-key" || password != "consumer-secret" {
				t.Error("OAuth request did not use configured basic credentials")
			}
			_, _ = w.Write([]byte(`{"access_token":"test-token"}`))
		case "/mpesa/stkpush/v1/processrequest":
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("STK request did not use the OAuth token")
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["Amount"] != float64(2450) || payload["PhoneNumber"] != "254712345678" || payload["CallBackURL"] != "https://halisi.example/api/mpesa/callback/token" {
				t.Errorf("unexpected STK payload: %#v", payload)
			}
			password, _ := payload["Password"].(string)
			decoded, err := base64.StdEncoding.DecodeString(password)
			timestamp, _ := payload["Timestamp"].(string)
			if err != nil || !strings.HasPrefix(string(decoded), "123456passkey") || !strings.HasSuffix(string(decoded), timestamp) || len(timestamp) != 14 || len(string(decoded)) != len("123456passkey20060102150405") {
				t.Errorf("invalid STK password field")
			}
			sawSTK = true
			_, _ = w.Write([]byte(`{"MerchantRequestID":"merchant-1","CheckoutRequestID":"checkout-1","ResponseCode":"0","ResponseDescription":"Accepted"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewDaraja("sandbox", "consumer-key", "consumer-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = server.URL
	ctx := context.Background()
	token, err := client.Authorize(ctx)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Initiate(ctx, token, STKRequest{
		ShortCode: "123456", Passkey: "passkey", TransactionType: "CustomerPayBillOnline",
		PhoneNumber: "254712345678", Amount: 2450, AccountReference: "HALISI-1", Description: "Order 1",
		CallbackURL: "https://halisi.example/api/mpesa/callback/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawSTK || response.CheckoutRequestID != "checkout-1" || response.MerchantRequestID != "merchant-1" {
		t.Fatalf("unexpected STK response: %+v", response)
	}
}

func TestDarajaRejectsUnacceptedSTKRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/v1/generate" {
			_, _ = w.Write([]byte(`{"access_token":"test-token"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ResponseCode":"1","ResponseDescription":"Rejected"}`))
	}))
	defer server.Close()
	client, err := NewDaraja("sandbox", "key", "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = server.URL
	token, err := client.Authorize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Initiate(context.Background(), token, STKRequest{ShortCode: "123456", Passkey: "pass", TransactionType: "CustomerPayBillOnline", PhoneNumber: "254712345678", Amount: 10, CallbackURL: "https://halisi.example/callback"})
	if err == nil {
		t.Fatal("expected rejected STK request to return an error")
	}
}

func TestDarajaMarksOAuthFailureAsNotSubmitted(t *testing.T) {
	var stkCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/v1/generate" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		stkCalls++
		http.Error(w, "unexpected STK request", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := NewDaraja("sandbox", "key", "bad-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = server.URL
	_, err = client.Authorize(context.Background())
	if !errors.Is(err, ErrSTKNotSubmitted) {
		t.Fatalf("OAuth failure error = %v, want ErrSTKNotSubmitted", err)
	}
	if stkCalls != 0 {
		t.Fatal("STK endpoint was called after OAuth failed")
	}
}
