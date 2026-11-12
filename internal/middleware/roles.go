package middleware

import (
	"github.com/wanjiru-noreen/halisi/internal/models"
	"net/http"
)

func RequirePermission(permission models.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r.Context())
		if !ok {
			authError(w, http.StatusUnauthorized, "Authentication required.")
			return
		}
		if !models.Allows(models.Role(user.Role), permission) {
			authError(w, http.StatusForbidden, "You do not have permission to access this resource.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
