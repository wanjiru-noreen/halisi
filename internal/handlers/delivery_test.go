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
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

func TestDeliveryManagementPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for delivery integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("delivery_test_%d", time.Now().UnixNano())
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
	if err := m.Steps(6); err != nil {
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
	reject := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err == nil {
			t.Fatalf("invalid operation accepted: %s", sql)
		}
	}
	users := map[string]int64{}
	tokens := map[string]string{}
	for actor, role := range map[string]string{"customer": "customer", "other": "customer", "vendor": "vendor", "otherVendor": "vendor", "rider": "rider", "otherRider": "rider", "pendingRider": "rider", "admin": "admin"} {
		users[actor] = id("INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,'test-hash',$3) RETURNING id", actor, actor+"@example.invalid", role)
		tokenBytes := sha256.Sum256([]byte(schema + actor))
		tokens[actor] = base64.RawURLEncoding.EncodeToString(tokenBytes[:])
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokens[actor])))
		exec("INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1,$2,NOW()+INTERVAL '1 hour')", users[actor], hash)
	}
	for _, actor := range []string{"rider", "otherRider", "pendingRider"} {
		status := "approved"
		if actor == "pendingRider" {
			status = "pending"
		}
		exec("INSERT INTO rider_profiles(user_id,full_name,phone,city,status) VALUES($1,$2,'+254712345678','Nairobi',$3)", users[actor], actor, status)
	}
	vendor := id("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Shop','Pickup street','approved',TRUE) RETURNING id", users["vendor"])
	address := id("INSERT INTO addresses(user_id,address_line,city,latitude,longitude) VALUES($1,'Dropoff street','Nairobi',-1.2,36.8) RETURNING id", users["customer"])
	otherAddress := id("INSERT INTO addresses(user_id,address_line,city) VALUES($1,'Other private street','Nairobi') RETURNING id", users["other"])
	order := id("INSERT INTO orders(customer_id,vendor_id,address_id,total,delivery_address) VALUES($1,$2,$3,1000,'Dropoff street, Nairobi') RETURNING id", users["customer"], vendor, address)
	backfilled := id("INSERT INTO orders(customer_id,vendor_id,address_id,status,total,delivery_address) VALUES($1,$2,$3,'confirmed',1000,'Historical dropoff') RETURNING id", users["customer"], vendor, address)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	var backfillCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE order_id=$1 AND status='pending' AND address_id=$2", backfilled, address).Scan(&backfillCount); err != nil || backfillCount != 1 {
		t.Fatal("eligible confirmed order was not backfilled", err)
	}
	authService, err := services.NewAuth(pool)
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(authService, "production")
	resources := handlers.NewResources(pool, auth)
	handler := routes.New(pool, resources.Routes)
	request := func(actor, method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if actor != "" {
			r.AddCookie(&http.Cookie{Name: "__Host-halisi_session", Value: tokens[actor]})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	check := func(actor, method, path string, want int) *httptest.ResponseRecorder {
		t.Helper()
		w := request(actor, method, path)
		if w.Code != want {
			t.Fatalf("%s %s as %s: got %d want %d: %s", method, path, actor, w.Code, want, w.Body.String())
		}
		return w
	}
	confirmPath := fmt.Sprintf("/api/vendor/orders/%d/confirm", order)
	check("", "POST", confirmPath, 401)
	check("customer", "POST", confirmPath, 403)
	check("otherVendor", "POST", confirmPath, 403)
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE order_id=$1", order).Scan(&count); err != nil || count != 0 {
		t.Fatal("pending order received a delivery", err)
	}
	reject("INSERT INTO deliveries(order_id) VALUES($1)", order)
	// Concurrent confirmation commits once and cannot create two deliveries.
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); codes <- request("vendor", "POST", confirmPath).Code }()
	}
	wg.Wait()
	close(codes)
	results := map[int]int{}
	for code := range codes {
		results[code]++
	}
	if results[200] != 1 || results[403] != 1 {
		t.Fatalf("concurrent confirmation: %v", results)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE order_id=$1", order).Scan(&count); err != nil || count != 1 {
		t.Fatal("confirmation did not create exactly one delivery", err)
	}
	var delivery dbsqlc.Delivery
	if err := pool.QueryRow(ctx, "SELECT id FROM deliveries WHERE order_id=$1", order).Scan(&delivery.ID); err != nil {
		t.Fatal(err)
	}
	reject("INSERT INTO deliveries(order_id) VALUES($1)", order)
	path := fmt.Sprintf("/api/customer/deliveries/%d", delivery.ID)
	check("", "GET", path, 401)
	check("other", "GET", path, 403)
	check("customer", "GET", "/api/customer/deliveries/999999", 403)
	check("customer", "GET", "/api/customer/deliveries/invalid", 422)
	check("otherVendor", "GET", fmt.Sprintf("/api/vendor/deliveries/%d", delivery.ID), 403)
	check("customer", "GET", fmt.Sprintf("/api/admin/deliveries/%d", delivery.ID), 403)
	for actor, prefix := range map[string]string{"customer": "customer", "vendor": "vendor", "admin": "admin"} {
		w := check(actor, "GET", fmt.Sprintf("/api/%s/deliveries/%d", prefix, delivery.ID), 200)
		var response struct {
			Delivery dbsqlc.Delivery `json:"delivery"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		d := response.Delivery
		if d.OrderID != order || d.CustomerID != users["customer"] || d.VendorID != vendor || d.AddressID.Int64 != address || d.Status != "pending" || d.RiderID.Valid || d.PickupAddress != "Pickup street" || d.DropoffAddress != "Dropoff street, Nairobi" || !d.DropoffLatitude.Valid {
			t.Fatalf("invalid delivery snapshot: %+v", d)
		}
	}
	exec("UPDATE addresses SET address_line='Changed location',latitude=0,longitude=0 WHERE id=$1", address)
	exec("UPDATE vendors SET address='Changed pickup' WHERE id=$1", vendor)
	w := check("customer", "GET", path, 200)
	var response struct {
		Delivery dbsqlc.Delivery `json:"delivery"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Delivery.PickupAddress != "Pickup street" || response.Delivery.DropoffAddress != "Dropoff street, Nairobi" {
		t.Fatal("location snapshots changed")
	}
	// Shop edits invalidate approval; restore the trusted fixture for later orders.
	exec("UPDATE vendors SET status='approved',is_active=TRUE WHERE id=$1", vendor)
	reject("UPDATE deliveries SET address_id=$2 WHERE id=$1", delivery.ID, otherAddress)
	reject("UPDATE deliveries SET customer_id=$2 WHERE id=$1", delivery.ID, users["other"])
	reject("UPDATE deliveries SET status='picked_up' WHERE id=$1", delivery.ID)
	reject("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE id=$1", delivery.ID, users["pendingRider"])
	reject("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE id=$1", delivery.ID, users["customer"])
	exec("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE id=$1", delivery.ID, users["rider"])
	check("rider", "GET", fmt.Sprintf("/api/rider/deliveries/%d", delivery.ID), 200)
	check("otherRider", "GET", fmt.Sprintf("/api/rider/deliveries/%d", delivery.ID), 403)
	exec("UPDATE rider_profiles SET status='suspended' WHERE user_id=$1", users["rider"])
	check("rider", "GET", fmt.Sprintf("/api/rider/deliveries/%d", delivery.ID), 403)
	exec("UPDATE rider_profiles SET status='approved' WHERE user_id=$1", users["rider"])
	reject("UPDATE deliveries SET status='delivered',delivered_at=NOW() WHERE id=$1", delivery.ID)
	exec("UPDATE deliveries SET status='picked_up' WHERE id=$1", delivery.ID)
	reject("UPDATE deliveries SET rider_id=$2 WHERE id=$1", delivery.ID, users["otherRider"])
	reject("UPDATE deliveries SET status='cancelled' WHERE id=$1", delivery.ID)
	exec("UPDATE deliveries SET status='delivered',delivered_at=NOW() WHERE id=$1", delivery.ID)
	reject("UPDATE deliveries SET status='assigned',delivered_at=NULL WHERE id=$1", delivery.ID)
	for _, terminal := range []string{"cancelled", "failed"} {
		terminalOrder := id("INSERT INTO orders(customer_id,vendor_id,address_id,status,total,delivery_address) VALUES($1,$2,$3,'confirmed',1000,'Street') RETURNING id", users["customer"], vendor, address)
		if terminal == "failed" {
			exec("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE order_id=$1", terminalOrder, users["rider"])
		}
		exec("UPDATE deliveries SET status=$2 WHERE order_id=$1", terminalOrder, terminal)
		reject("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE order_id=$1", terminalOrder, users["rider"])
	}
	// Invalid eligibility must roll back the order confirmation too.
	missingAddress := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,1000,'Legacy') RETURNING id", users["customer"], vendor)
	check("vendor", "POST", fmt.Sprintf("/api/vendor/orders/%d/confirm", missingAddress), 403)
	reject("UPDATE orders SET status='confirmed' WHERE id=$1", missingAddress)
	inactiveOrder := id("INSERT INTO orders(customer_id,vendor_id,address_id,total,delivery_address) VALUES($1,$2,$3,1000,'Street') RETURNING id", users["customer"], vendor, address)
	exec("UPDATE vendors SET is_active=FALSE WHERE id=$1", vendor)
	check("vendor", "POST", fmt.Sprintf("/api/vendor/orders/%d/confirm", inactiveOrder), 403)
	reject("UPDATE orders SET status='confirmed' WHERE id=$1", inactiveOrder)
	var state string
	if err := pool.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", inactiveOrder).Scan(&state); err != nil || state != "pending" {
		t.Fatal("failed creation did not roll back confirmation", err)
	}
}
