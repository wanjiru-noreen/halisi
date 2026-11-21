package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkStorefrontListProductsValidation(b *testing.B) {
	s := handlers.NewStorefront(nil)
	mux := http.NewServeMux()
	s.Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/storefront/products?q=gas&vendor_id=invalid", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkCheckoutValidationOversized(b *testing.B) {
	c := handlers.NewCheckout(nil, handlers.NewAuth(nil, "test"))
	mux := http.NewServeMux()
	c.Routes(mux)

	oversizedPayload := `{"address_id":1,"items":[{"product_id":1,"quantity":` + strings.Repeat("9", 5000) + `}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/customer/checkout", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.WithCurrentUser(req.Context(), services.User{ID: 10, Role: string(models.Customer)})
	req = req.WithContext(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Body = io.NopCloser(strings.NewReader(oversizedPayload))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}
