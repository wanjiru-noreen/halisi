package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

func TestCheckoutValidation(t *testing.T) {
	c := &Checkout{}

	tests := []struct {
		name        string
		contentType string
		body        string
		withUser    bool
		wantStatus  int
		wantError   string
	}{
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `{}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantError:   "Content-Type must be application/json",
		},
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{invalid}`,
			wantStatus:  http.StatusBadRequest,
			wantError:   "Invalid JSON request",
		},
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[],"unknown":true}`,
			wantStatus:  http.StatusBadRequest,
			wantError:   "Invalid JSON request",
		},
		{
			name:        "missing address",
			contentType: "application/json",
			body:        `{"address_id":0,"items":[{"product_id":1,"quantity":1}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Choose a delivery address and between 1 and 50 products",
		},
		{
			name:        "negative address id",
			contentType: "application/json",
			body:        `{"address_id":-5,"items":[{"product_id":1,"quantity":1}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Choose a delivery address and between 1 and 50 products",
		},
		{
			name:        "empty items",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Choose a delivery address and between 1 and 50 products",
		},
		{
			name:        "invalid product id",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":0,"quantity":1}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Product quantities must be between 1 and 100",
		},
		{
			name:        "negative product id",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":-1,"quantity":1}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Product quantities must be between 1 and 100",
		},
		{
			name:        "zero quantity",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":1,"quantity":0}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Product quantities must be between 1 and 100",
		},
		{
			name:        "excessive quantity",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":1,"quantity":101}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Product quantities must be between 1 and 100",
		},
		{
			name:        "duplicate product id",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":1,"quantity":2},{"product_id":1,"quantity":3}]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantError:   "Each product can appear only once in the cart",
		},
		{
			name:        "unauthenticated user",
			contentType: "application/json",
			body:        `{"address_id":1,"items":[{"product_id":1,"quantity":2}]}`,
			withUser:    false,
			wantStatus:  http.StatusUnauthorized,
			wantError:   "Authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/customer/checkout", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			if tt.withUser {
				ctx := middleware.WithCurrentUser(req.Context(), services.User{ID: 10, Role: string(models.Customer)})
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			c.submit(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantError != "" && !strings.Contains(rec.Body.String(), tt.wantError) {
				t.Fatalf("body %q does not contain expected error %q", rec.Body.String(), tt.wantError)
			}
		})
	}
}

func TestCheckoutCartRoutes(t *testing.T) {
	c := NewCheckout(nil, NewAuth(nil, "test"))
	mux := http.NewServeMux()
	c.Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/customer/cart", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
