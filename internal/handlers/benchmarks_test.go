package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
)

func BenchmarkNormalizeMPesaPhone(b *testing.B) {
	numbers := []string{
		"0712 345 678",
		"+254 (712) 345678",
		"254712345678",
		"0112-345-678",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = handlers.NormalizeMPesaPhone(numbers[i%len(numbers)])
	}
}

func BenchmarkRouteDispatch(b *testing.B) {
	handler := routes.New(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkStorefrontListProductsValidation(b *testing.B) {
	s := handlers.NewStorefront(nil)
	mux := http.NewServeMux()
	s.Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?q=gas&vendor_id=invalid", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkCheckoutValidationOversized(b *testing.B) {
	c := handlers.NewCheckout(nil, handlers.NewAuth(nil, "test"))
	mux := http.NewServeMux()
	c.Routes(mux)

	oversizedPayload := `{"address_id":1,"items":[{"product_id":1,"quantity":` + strings.Repeat("9", 5000) + `}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/customer/checkout", strings.NewReader(oversizedPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rec, req)
	}
}
