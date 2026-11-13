package handlers_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

func TestAdminDashboardPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for administrator dashboard integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("admin_dashboard_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid test URL")
	}
	params := u.Query()
	params.Set("search_path", schema)
	u.RawQuery = params.Encode()
	m, err := database.NewMigrator(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	id := func(sql string, args ...any) int64 {
		t.Helper()
		var value int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	users := map[string]int64{}
	tokens := map[string]string{}
	for actor, role := range map[string]string{"admin": "admin", "customer": "customer", "outside": "customer", "vendor": "vendor", "pendingVendor": "vendor", "rider": "rider", "pendingRider": "rider"} {
		created := "2026-02-01T10:00:00Z"
		if actor == "outside" {
			created = "2026-01-31T20:59:59Z"
		}
		users[actor] = id("INSERT INTO users(name,email,password_hash,role,created_at) VALUES($1,$2,'private-password-hash',$3,$4) RETURNING id", actor, actor+"@example.invalid", role, created)
		raw := sha256.Sum256([]byte(schema + actor))
		tokens[actor] = base64.RawURLEncoding.EncodeToString(raw[:])
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokens[actor])))
		exec("INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1,$2,NOW()+INTERVAL '1 hour')", users[actor], hash)
	}
	vendor := id("INSERT INTO vendors(user_id,name,address,status,is_active,created_at) VALUES($1,'Live shop','Private shop address','approved',TRUE,'2026-02-01T10:00:00Z') RETURNING id", users["vendor"])
	id("INSERT INTO vendors(user_id,name,address,created_at) VALUES($1,'Pending shop','Private address','2025-01-01T00:00:00Z') RETURNING id", users["pendingVendor"])
	exec("INSERT INTO rider_profiles(user_id,full_name,phone,city,status,created_at) VALUES($1,'Approved Rider','+254712345678','Nairobi','approved','2026-02-01T10:00:00Z'),($2,'Pending Rider','+254712345679','Nairobi','pending','2025-01-01T00:00:00Z')", users["rider"], users["pendingRider"])
	authService, err := services.NewAuth(pool)
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(authService, "production")
	resources := handlers.NewResources(pool, auth)
	handler := routes.New(pool, auth.Routes, resources.Routes)
	request := func(actor, path string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("X-Role", "admin")
		if actor != "" {
			r.AddCookie(&http.Cookie{Name: "__Host-halisi_session", Value: tokens[actor]})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("GET %s as %s: got %d want %d: %s", path, actor, w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("dashboard response was cacheable")
		}
		return w
	}
	const path = "/api/admin/dashboard?start=2026-02-01&end=2026-02-02"
	request("", path, 401)
	request("", "/admin/dashboard", 401)
	for _, actor := range []string{"customer", "vendor", "rider", "pendingRider"} {
		request(actor, path, 403)
		request(actor, "/admin/dashboard", 403)
	}
	page := request("admin", "/admin/dashboard", 200)
	if !strings.Contains(page.Body.String(), "admin-dashboard.js") || !strings.Contains(page.Body.String(), "Platform overview") {
		t.Fatal("administrator dashboard template was not served")
	}
	request("admin", "/api/admin/dashboard?start=2026-02-30&end=2026-03-01", 422)
	request("admin", "/api/admin/dashboard?start=2026-02-01", 422)
	decode := func(w *httptest.ResponseRecorder) services.Dashboard {
		t.Helper()
		var result services.Dashboard
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	empty := decode(request("admin", path, 200))
	if empty.Orders.Total != 0 || len(empty.Financial.Currencies) != 0 || empty.Financial.Commissions != nil || empty.Financial.Settlements != nil {
		t.Fatal("empty financial data was fabricated")
	}
	// More than a listing page of customers: totals must aggregate the full table.
	exec("INSERT INTO users(name,email,password_hash,role,created_at) SELECT 'Private customer', 'bulk-'||i||'@example.invalid','private-password-hash','customer','2026-02-01T10:00:00Z' FROM generate_series(1,105) i")
	order := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address,created_at) VALUES($1,$2,1000,'Private customer address','2026-01-31T21:00:00Z') RETURNING id", users["customer"], vendor)
	usdOrder := id("INSERT INTO orders(customer_id,vendor_id,total,currency,delivery_address,created_at) VALUES($1,$2,200,'USD','Private address','2026-02-02T20:59:59Z') RETURNING id", users["customer"], vendor)
	id("INSERT INTO orders(customer_id,vendor_id,total,status,delivery_address,created_at) VALUES($1,$2,100,'cancelled','Private address','2026-02-01T11:00:00Z') RETURNING id", users["customer"], vendor)
	id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address,created_at) VALUES($1,$2,100,'Private address','2026-02-02T21:00:00Z') RETURNING id", users["customer"], vendor)
	for _, p := range []struct{ amount, status, date string }{
		{"100.10", "completed", "2026-01-31T21:00:00Z"}, {"0.20", "completed", "2026-02-01T11:00:00Z"},
		{"40.15", "refunded", "2026-02-02T20:59:59Z"}, {"999.99", "pending", "2026-02-01T11:00:00Z"},
		{"500.00", "failed", "2026-02-01T11:00:00Z"}, {"10000.00", "completed", "2026-01-31T20:59:59Z"},
		{"20000.00", "refunded", "2026-02-02T21:00:00Z"},
	} {
		exec("INSERT INTO payments(order_id,amount,status,created_at) VALUES($1,$2,$3,$4)", order, p.amount, p.status, p.date)
	}
	exec("INSERT INTO payments(order_id,amount,currency,status,created_at) VALUES($1,10.50,'USD','completed','2026-02-01T11:00:00Z')", usdOrder)
	w := request("admin", path, 200)
	data := decode(w)
	if data.Accounts.Customers.Total != 107 || data.Accounts.Customers.Added != 106 || data.Accounts.Vendors.Total != 2 || data.Accounts.Vendors.Added != 1 || data.Accounts.Vendors.Approved != 1 || data.Accounts.Vendors.Pending != 1 || data.Accounts.Riders.Total != 2 || data.Accounts.Riders.Added != 1 || data.Accounts.Riders.Pending != 1 || data.Accounts.Riders.Approved != 1 {
		t.Fatalf("incorrect account totals: %+v", data.Accounts)
	}
	if data.Orders.Total != 3 {
		t.Fatalf("order totals multiplied by joins or dates incorrect: %+v", data.Orders)
	}
	if len(data.Orders.ByStatus) != 2 || data.Orders.ByStatus[0].Status != "cancelled" || data.Orders.ByStatus[0].Count != 1 || data.Orders.ByStatus[1].Count != 2 {
		t.Fatalf("incorrect statuses: %+v", data.Orders)
	}
	if len(data.Financial.Currencies) != 2 {
		t.Fatal("currencies were combined")
	}
	kes, usd := data.Financial.Currencies[0], data.Financial.Currencies[1]
	if kes.Currency != "KES" || kes.PaymentCount != 3 || kes.GrossSales != "140.45" || kes.Refunds != "40.15" || kes.NetCollections != "100.30" || usd.Currency != "USD" || usd.GrossSales != "10.50" || usd.NetCollections != "10.50" {
		t.Fatalf("incorrect financial totals: %+v", data.Financial)
	}
	if data.Period.Timezone != "Africa/Nairobi" || len(data.Activity) != 20 || data.GeneratedAt.IsZero() {
		t.Fatal("missing period/activity metadata")
	}
	start := time.Date(2026, 1, 31, 21, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	for i, event := range data.Activity {
		if event.OccurredAt.Time.Before(start) || !event.OccurredAt.Time.Before(end) {
			t.Fatal("activity outside inclusive local date range")
		}
		if i > 0 && event.OccurredAt.Time.After(data.Activity[i-1].OccurredAt.Time) {
			t.Fatal("activity not sorted most recent first")
		}
	}
	for _, secret := range []string{"password_hash", "private-password-hash", "token_hash", "Private customer address", "example.invalid"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("dashboard exposed private detail %q", secret)
		}
	}
	id("INSERT INTO users(name,email,password_hash,role,created_at) VALUES('Live customer','live@example.invalid','test-hash','customer','2026-02-01T12:00:00Z') RETURNING id")
	if got := decode(request("admin", path, 200)); got.Accounts.Customers.Total != 108 {
		t.Fatal("dashboard did not reflect new database records")
	}
	exec("UPDATE users SET role='customer' WHERE id=$1", users["admin"])
	request("admin", path, 403)
	request("admin", "/admin/dashboard", 403)
	exec("UPDATE users SET role='admin' WHERE id=$1", users["admin"])
	// Authentication still works while a dashboard query fails. Failed read-only
	// transactions must roll back and allow the next dashboard request to recover.
	exec("ALTER TABLE payments RENAME TO unavailable_payments")
	failed := request("admin", path, 503)
	if strings.Contains(failed.Body.String(), "payments") || strings.Contains(failed.Body.String(), "SELECT") {
		t.Fatal("dashboard failure leaked query details")
	}
	exec("ALTER TABLE unavailable_payments RENAME TO payments")
	request("admin", path, 200)
	pool.Close()
	unavailable := request("admin", path, 503)
	if strings.Contains(unavailable.Body.String(), "postgres") {
		t.Fatal("database failure leaked internal details")
	}
}
