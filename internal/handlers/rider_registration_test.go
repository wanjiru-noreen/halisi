package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"golang.org/x/crypto/bcrypt"
)

func TestRiderRegistrationPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for rider registration integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	adminDB, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("cannot initialize test pool")
	}
	defer adminDB.Close()
	schema := fmt.Sprintf("rider_registration_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := adminDB.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("test schema cleanup failed")
		}
	}()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid test URL")
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	m, err := database.NewMigrator(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Steps(6); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	insertID := func(sql string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	const password = "rider registration test password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), services.PasswordCost)
	if err != nil {
		t.Fatal(err)
	}
	authService, err := services.NewAuth(pool)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	cookies := map[string]*http.Cookie{}
	roleByActor := map[string]string{"applicant": "customer", "other": "customer", "legacy": "rider", "vendor": "vendor", "admin": "admin"}
	for actor, role := range roleByActor {
		ids[actor] = insertID("INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,$4) RETURNING id", actor, actor+"@example.invalid", string(hash), role)
		session, err := authService.Login(ctx, actor+"@example.invalid", password, "")
		if err != nil {
			t.Fatal(err)
		}
		cookies[actor] = &http.Cookie{Name: "__Host-halisi_session", Value: session.Token}
	}
	vendor := insertID("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Rider fixture shop','Nairobi','approved',TRUE) RETURNING id", ids["vendor"])
	order := insertID("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,100,'Nairobi') RETURNING id", ids["applicant"], vendor)
	delivery := insertID("INSERT INTO deliveries(order_id,rider_id,status) VALUES($1,$2,'assigned') RETURNING id", order, ids["legacy"])
	// Preserve a pre-migration assignment to an unverified legacy rider.
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(authService, "production")
	riders := services.NewRiderRegistration(pool)
	handler := handlers.NewRiderRegistration(auth, riders)
	resources := handlers.NewResources(pool, auth)
	server := httptest.NewTLSServer(routes.New(pool, auth.Routes, handler.Routes, resources.Routes))
	defer server.Close()
	client := server.Client()
	call := func(actor, method, path string, input any) (int, []byte, error) {
		var reader io.Reader
		if input != nil {
			data, err := json.Marshal(input)
			if err != nil {
				return 0, nil, err
			}
			reader = bytes.NewReader(data)
		}
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, reader)
		if err != nil {
			return 0, nil, err
		}
		if cookies[actor] != nil {
			r.AddCookie(cookies[actor])
		}
		r.Header.Set("Accept", "text/html")
		r.Header.Set("X-Role", "rider")
		if input != nil {
			r.Header.Set("Content-Type", "application/json")
		}
		response, err := client.Do(r)
		if err != nil {
			return 0, nil, err
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		return response.StatusCode, data, err
	}
	request := func(actor, method, path string, input any, want int) []byte {
		t.Helper()
		status, data, err := call(actor, method, path, input)
		if err != nil {
			t.Fatal(err)
		}
		if status != want {
			t.Fatalf("%s %s %s got %d, want %d: %s", actor, method, path, status, want, data)
		}
		return data
	}
	profile := services.RiderInput{FullName: " Alex Kamau ", Phone: " +254712345678 ", City: " Nairobi "}
	for _, method := range []string{"GET", "POST", "PATCH"} {
		request("", method, "/api/rider/registration", profile, 401)
	}
	request("applicant", "GET", "/api/rider/registration", nil, 404)
	request("other", "POST", "/api/rider/registration", services.RiderInput{FullName: " ", Phone: "bad", City: "Nairobi"}, 422)
	body := request("applicant", "POST", "/api/rider/registration", profile, 201)
	var saved struct {
		Rider dbsqlc.RiderProfile `json:"rider"`
		Email string              `json:"email"`
	}
	if err := json.Unmarshal(body, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Rider.Status != "pending" || saved.Rider.UserID != ids["applicant"] || saved.Rider.FullName != "Alex Kamau" || saved.Rider.Phone != "+254712345678" || saved.Email != "applicant@example.invalid" {
		t.Fatal("registration was not normalized, private or pending")
	}
	originalID := saved.Rider.ID
	request("applicant", "POST", "/api/rider/registration", profile, 409)
	for _, spoof := range []map[string]any{{"role": "admin"}, {"status": "approved"}, {"user_id": ids["applicant"]}, {"email": "someone@example.invalid"}} {
		spoof["full_name"] = "Other rider"
		spoof["phone"] = "+254712345678"
		spoof["city"] = "Nairobi"
		request("other", "POST", "/api/rider/registration", spoof, 400)
	}
	request("other", "GET", fmt.Sprintf("/api/rider/registration?user_id=%d", ids["applicant"]), nil, 404)
	request("other", "PATCH", "/api/rider/registration", map[string]string{"city": "Mombasa"}, 403)
	body = request("applicant", "PATCH", "/api/rider/registration", map[string]string{"city": "Nakuru"}, 200)
	if err := json.Unmarshal(body, &saved); err != nil || saved.Rider.City != "Nakuru" || saved.Rider.Phone != "+254712345678" || saved.Rider.Status != "pending" || saved.Rider.ID != originalID || !saved.Rider.UpdatedAt.Time.After(saved.Rider.CreatedAt.Time) {
		t.Fatal("pending profile update was not isolated or timestamped")
	}
	request("applicant", "PATCH", "/api/rider/registration", map[string]string{"phone": "bad"}, 422)
	request("applicant", "PATCH", "/api/rider/registration", map[string]string{}, 422)
	request("applicant", "PATCH", "/api/rider/registration", map[string]string{"status": "approved"}, 400)
	var role string
	if err := pool.QueryRow(ctx, "SELECT role FROM users WHERE id=$1", ids["applicant"]).Scan(&role); err != nil || role != "customer" {
		t.Fatal("registration changed the account role")
	}
	request("applicant", "GET", "/api/rider/deliveries", nil, 403)
	// Uniqueness must hold for concurrent registration, not just a preflight check.
	var wg sync.WaitGroup
	outcomes := make(chan int, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _, err := call("other", "POST", "/api/rider/registration", profile)
			if err != nil {
				failures <- err
			}
			outcomes <- status
		}()
	}
	wg.Wait()
	close(outcomes)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	counts := map[int]int{}
	for status := range outcomes {
		counts[status]++
	}
	if counts[201] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent duplicate was not rejected: %v", counts)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM rider_profiles WHERE user_id=$1", ids["other"]).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate application persisted")
	}
	body = request("other", "GET", fmt.Sprintf("/api/rider/registration?user_id=%d", ids["applicant"]), nil, 200)
	if err := json.Unmarshal(body, &saved); err != nil || saved.Rider.UserID != ids["other"] {
		t.Fatal("another user's profile was exposed")
	}
	for _, actor := range []string{"vendor", "admin"} {
		request(actor, "POST", "/api/rider/registration", profile, 201)
		if err := pool.QueryRow(ctx, "SELECT role FROM users WHERE id=$1", ids[actor]).Scan(&role); err != nil || role != roleByActor[actor] {
			t.Fatal("application changed a privileged role")
		}
		request(actor, "GET", "/api/rider/deliveries", nil, 403)
	}
	// Even an existing rider role and assigned row confer no access without approval.
	deliveryPath := fmt.Sprintf("/api/rider/deliveries/%d", delivery)
	request("legacy", "GET", deliveryPath, nil, 403)
	request("legacy", "POST", "/api/rider/registration", profile, 201)
	request("legacy", "GET", deliveryPath, nil, 403)
	request("legacy", "PATCH", deliveryPath, map[string]string{"status": "picked_up"}, 403)
	var deliveryStatus string
	if err := pool.QueryRow(ctx, "SELECT status FROM deliveries WHERE id=$1", delivery).Scan(&deliveryStatus); err != nil || deliveryStatus != "assigned" {
		t.Fatal("pending applicant changed a delivery")
	}
	// Trusted fixtures represent future approval; no approval HTTP API is added in 6.1.
	if _, err := pool.Exec(ctx, "UPDATE rider_profiles SET status='approved' WHERE user_id=$1", ids["legacy"]); err != nil {
		t.Fatal(err)
	}
	request("legacy", "GET", deliveryPath, nil, 200)
	request("legacy", "PATCH", "/api/rider/registration", map[string]string{"city": "Mombasa"}, 403)
	if _, err := pool.Exec(ctx, "UPDATE rider_profiles SET status='suspended' WHERE user_id=$1", ids["legacy"]); err != nil {
		t.Fatal(err)
	}
	request("legacy", "GET", deliveryPath, nil, 403)
	if _, err := pool.Exec(ctx, "UPDATE rider_profiles SET status='approved' WHERE user_id=$1", ids["applicant"]); err != nil {
		t.Fatal(err)
	}
	request("applicant", "GET", "/api/rider/deliveries", nil, 403)
}
