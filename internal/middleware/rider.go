package middleware

import (
	"context"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"net/http"
	"time"
)

type RiderApproval interface {
	RiderApproved(context.Context, int64) (bool, error)
}

func RequireApprovedRider(check RiderApproval, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r.Context())
		if !ok {
			authError(w, 401, "Authentication required.")
			return
		}
		if models.Role(user.Role) != models.Rider {
			authError(w, 403, "Approved rider access is required.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		approved, err := check.RiderApproved(ctx, user.ID)
		if err != nil {
			authError(w, 503, "Rider authorization is temporarily unavailable.")
			return
		}
		if !approved {
			authError(w, 403, "Rider approval is required before accessing deliveries.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
