package middleware

import (
	"context"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roleAuthenticator struct{ role string }

func (a roleAuthenticator) Authenticate(context.Context, string) (services.User, error) {
	return services.User{ID: 17, Role: a.role}, nil
}

func TestRolePermissionMatrix(t *testing.T) {
	for _, role := range []models.Role{models.Customer, models.Vendor, models.Rider, models.Administrator, "unknown", ""} {
		for _, permission := range []models.Permission{models.CustomerResources, models.VendorResources, models.RiderAssignments, models.Administration, "unknown"} {
			t.Run(string(role)+"/"+string(permission), func(t *testing.T) {
				next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					user, ok := CurrentUser(r.Context())
					if !ok || user.ID != 17 {
						t.Error("missing principal")
					}
					w.WriteHeader(204)
				})
				handler := RequireAuth(roleAuthenticator{role: string(role)}, "session", RequirePermission(permission, next))
				r := httptest.NewRequest("GET", "/api/protected", nil)
				r.AddCookie(&http.Cookie{Name: "session", Value: "fixture"})
				r.Header.Set("X-Role", "admin")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				permitted := (role == models.Customer && permission == models.CustomerResources) || (role == models.Vendor && permission == models.VendorResources) || (role == models.Rider && permission == models.RiderAssignments) || (role == models.Administrator && permission == models.Administration)
				want := 403
				if permitted {
					want = 204
				}
				if w.Code != want {
					t.Fatalf("got %d, want %d", w.Code, want)
				}
			})
		}
	}
}

func TestAPIAuthorizationWithoutAuthentication(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthenticated handler reached") })
	for _, handler := range []http.Handler{RequirePermission(models.Administration, next), RequireAuth(roleAuthenticator{role: "admin"}, "session", RequirePermission(models.Administration, next))} {
		r := httptest.NewRequest("GET", "/api/admin/users", nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("API authentication returned %d, want 401", w.Code)
		}
	}
}
