package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type approvalStub struct {
	approved bool
	err      error
	calls    int
}

func (s *approvalStub) RiderApproved(ctx context.Context, id int64) (bool, error) {
	s.calls++
	if id != 17 {
		panic("incorrect principal")
	}
	return s.approved, s.err
}
func TestRiderApprovalGate(t *testing.T) {
	for _, tt := range []struct {
		name, role  string
		approved    bool
		err         error
		want, calls int
	}{
		{"approved rider", "rider", true, nil, 204, 1}, {"pending or absent profile", "rider", false, nil, 403, 1}, {"approved customer has no rider role", "customer", true, nil, 403, 0}, {"database unavailable", "rider", false, errors.New("private database error"), 503, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			check := &approvalStub{approved: tt.approved, err: tt.err}
			handler := RequireAuth(roleAuthenticator{role: tt.role}, "session", RequireApprovedRider(check, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
			r := httptest.NewRequest("GET", "/api/rider/deliveries", nil)
			r.AddCookie(&http.Cookie{Name: "session", Value: "fixture"})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tt.want || check.calls != tt.calls {
				t.Fatalf("got %d with %d checks", w.Code, check.calls)
			}
			if tt.err != nil && w.Body.String() == tt.err.Error() {
				t.Fatal("database error exposed")
			}
		})
	}
	w := httptest.NewRecorder()
	RequireApprovedRider(&approvalStub{}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthenticated access") })).ServeHTTP(w, httptest.NewRequest("GET", "/api/rider/deliveries", nil))
	if w.Code != 401 {
		t.Fatal("missing authentication was not rejected")
	}
}
