package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

type fakeSTKClient struct{}

func (f *fakeSTKClient) Authorize(context.Context) (string, error) {
	return "test_bearer_token", nil
}

func (f *fakeSTKClient) Initiate(context.Context, string, services.STKRequest) (services.STKResponse, error) {
	return services.STKResponse{
		ResponseCode:      "0",
		MerchantRequestID: "merchant_123",
		CheckoutRequestID: "checkout_123",
	}, nil
}

func TestNormalizeMPesaPhone(t *testing.T) {
	for raw, want := range map[string]string{
		"0712 345 678":      "254712345678",
		"+254 (712) 345678": "254712345678",
		"254712345678":      "254712345678",
		"0112-345-678":      "254112345678",
	} {
		got, err := handlers.NormalizeMPesaPhone(raw)
		if err != nil || got != want {
			t.Errorf("NormalizeMPesaPhone(%q) = (%q, %v), want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "12345", "071234567", "07123456789", "+254912345678"} {
		if _, err := handlers.NormalizeMPesaPhone(raw); err == nil {
			t.Errorf("NormalizeMPesaPhone(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestMPesaAvailability(t *testing.T) {
	t.Run("disabled when client is nil", func(t *testing.T) {
		mpesa := handlers.NewMPesa(nil, handlers.NewAuth(&fakeAuth{}, "test"), nil, "https://example.com", "", "", "")
		handler := routes.New(nil, mpesa.Routes)

		req := httptest.NewRequest(http.MethodGet, "/api/mpesa/availability", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"enabled":false`) {
			t.Fatalf("expected enabled:false, got: %s", rec.Body.String())
		}
	})

	t.Run("enabled when client is configured", func(t *testing.T) {
		mpesa := handlers.NewMPesa(nil, handlers.NewAuth(&fakeAuth{}, "test"), &fakeSTKClient{}, "https://example.com", "174379", "passkey", "CustomerPayBillOnline")
		handler := routes.New(nil, mpesa.Routes)

		req := httptest.NewRequest(http.MethodGet, "/api/mpesa/availability", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"enabled":true`) {
			t.Fatalf("expected enabled:true, got: %s", rec.Body.String())
		}
	})
}

func TestMPesaCallbackValidation(t *testing.T) {
	mpesa := handlers.NewMPesa(nil, handlers.NewAuth(&fakeAuth{}, "test"), &fakeSTKClient{}, "https://example.com", "174379", "passkey", "CustomerPayBillOnline")
	handler := routes.New(nil, mpesa.Routes)

	tests := []struct {
		name       string
		token      string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			token:      "tok123",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed JSON",
			token:      "tok123",
			body:       `{malformed}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing ResultCode",
			token:      "tok123",
			body:       `{"Body":{"stkCallback":{"MerchantRequestID":"m1","CheckoutRequestID":"c1"}}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing CheckoutRequestID",
			token:      "tok123",
			body:       `{"Body":{"stkCallback":{"MerchantRequestID":"m1","ResultCode":0}}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing MerchantRequestID",
			token:      "tok123",
			body:       `{"Body":{"stkCallback":{"CheckoutRequestID":"c1","ResultCode":0}}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "oversized CheckoutRequestID",
			token:      "tok123",
			body:       `{"Body":{"stkCallback":{"MerchantRequestID":"m1","CheckoutRequestID":"` + strings.Repeat("a", 129) + `","ResultCode":0}}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/mpesa/callback/"+tt.token, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
