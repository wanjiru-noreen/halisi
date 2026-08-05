package middleware

import "net/http"

func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		session, err := Store.Get(r, "halisi-session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok || userRole != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
