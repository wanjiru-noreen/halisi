package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStorefrontValidation(t *testing.T) {
	s := &Storefront{}

	t.Run("listProducts rejects long search query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?q="+strings.Repeat("a", 101), nil)
		rec := httptest.NewRecorder()
		s.listProducts(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("listProducts rejects invalid vendor_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?vendor_id=invalid", nil)
		rec := httptest.NewRecorder()
		s.listProducts(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("listProducts rejects negative vendor_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?vendor_id=-1", nil)
		rec := httptest.NewRecorder()
		s.listProducts(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("listProducts rejects zero vendor_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?vendor_id=0", nil)
		rec := httptest.NewRecorder()
		s.listProducts(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("listVendors rejects long search query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storefront/vendors?q="+strings.Repeat("a", 101), nil)
		rec := httptest.NewRecorder()
		s.listVendors(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})
}

func TestStorefrontPages(t *testing.T) {
	s := NewStorefront(nil)
	mux := http.NewServeMux()
	s.Routes(mux)

	for _, path := range []string{"/customer/vendors", "/customer/products"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
		})
	}
}
