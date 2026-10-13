package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		sessionValues  map[interface{}]interface{}
		requiredRole   string
		expectedStatus int
	}{
		{
			name: "allows correct role",
			sessionValues: map[interface{}]interface{}{
				"user_id": 1,
				"role":    "customer",
			},
			requiredRole:   "customer",
			expectedStatus: http.StatusOK,
		},
		{
			name: "rejects wrong role",
			sessionValues: map[interface{}]interface{}{
				"user_id": 1,
				"role":    "owner",
			},
			requiredRole:   "customer",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "rejects missing user id",
			sessionValues: map[interface{}]interface{}{
				"role": "customer",
			},
			requiredRole:   "customer",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name: "rejects missing role",
			sessionValues: map[interface{}]interface{}{
				"user_id": 1,
			},
			requiredRole:   "customer",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			)

			rec := httptest.NewRecorder()

			session, err := Store.Get(req, "halisi-session")
			if err != nil {
				t.Fatalf("failed to get session: %v", err)
			}

			for key, value := range tt.sessionValues {
				session.Values[key] = value
			}

			if err := session.Save(req, rec); err != nil {
				t.Fatalf("failed to save session: %v", err)
			}

			req.Header.Set(
				"Cookie",
				rec.Header().Get("Set-Cookie"),
			)

			handler := RequireRole(
				tt.requiredRole,
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			)

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}
