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
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"golang.org/x/crypto/bcrypt"
)

func TestRoleResourcesPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for role/ownership integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("cannot initialize test pool")
	}
	defer admin.Close()
	schema := fmt.Sprintf("roles_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("test cleanup failed")
		}
	}()
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid test URL")
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	m, err := database.NewMigrator(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatal("test migrations failed")
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal("cannot initialize schema pool")
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
	const password = "roles test passphrase"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), services.PasswordCost)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"c1": "customer", "c2": "customer", "v1": "vendor", "v2": "vendor", "r1": "rider", "r2": "rider", "admin": "admin"}
	users := map[string]int64{}
	for name, role := range roles {
		users[name] = id("INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,$4) RETURNING id", name, name+"@example.invalid", string(hash), role)
	}
	v1 := id("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Shop one','Nairobi','approved',TRUE) RETURNING id", users["v1"])
	v2 := id("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Shop two','Mombasa','approved',TRUE) RETURNING id", users["v2"])
	p1 := id("INSERT INTO products(vendor_id,name,price) VALUES($1,'Cylinder one',1000) RETURNING id", v1)
	p2 := id("INSERT INTO products(vendor_id,name,price) VALUES($1,'Cylinder two',2000) RETURNING id", v2)
	a1 := id("INSERT INTO addresses(user_id,address_line,city) VALUES($1,'Private one','Nairobi') RETURNING id", users["c1"])
	a2 := id("INSERT INTO addresses(user_id,address_line,city) VALUES($1,'Private two','Mombasa') RETURNING id", users["c2"])
	o1 := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,1000,'Private one') RETURNING id", users["c1"], v1)
	o2 := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,2000,'Private two') RETURNING id", users["c2"], v2)
	payment1 := id("INSERT INTO payments(order_id,amount) VALUES($1,1000) RETURNING id", o1)
	payment2 := id("INSERT INTO payments(order_id,amount) VALUES($1,2000) RETURNING id", o2)
	d1 := id("INSERT INTO deliveries(order_id,rider_id,status) VALUES($1,$2,'assigned') RETURNING id", o1, users["r1"])
	d2 := id("INSERT INTO deliveries(order_id,rider_id,status) VALUES($1,$2,'assigned') RETURNING id", o2, users["r2"])
	o3 := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,100,'Unassigned') RETURNING id", users["c2"], v2)
	unassigned := id("INSERT INTO deliveries(order_id) VALUES($1) RETURNING id", o3)
	service, err := services.NewAuth(pool, services.WithVerification(&captureMailer{}, "https://halisi.example"))
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(service, "production")
	resources := handlers.NewResources(pool, auth)
	server := httptest.NewTLSServer(routes.New(pool, auth.Routes, resources.Routes))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	cookies := map[string]*http.Cookie{}
	request := func(actor, method, path string, body any, want int) (*http.Response, []byte) {
		t.Helper()
		var reader io.Reader
		if body != nil {
			encoded, e := json.Marshal(body)
			if e != nil {
				t.Fatal(e)
			}
			reader = bytes.NewReader(encoded)
		}
		r, e := http.NewRequestWithContext(ctx, method, server.URL+path, reader)
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Accept", "text/html")
		r.Header.Set("X-Role", "admin")
		if body != nil {
			r.Header.Set("Content-Type", "application/json")
		}
		if cookie := cookies[actor]; cookie != nil {
			r.AddCookie(cookie)
		}
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		data, e := io.ReadAll(response.Body)
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s %s returned %d, want %d: %s", actor, method, path, response.StatusCode, want, data)
		}
		if strings.Contains(string(data), "password_hash") || strings.Contains(string(data), "token_hash") || strings.Contains(string(data), password) {
			t.Fatal("private credentials exposed")
		}
		return response, data
	}
	for name, role := range roles {
		response, body := request("", "POST", "/api/auth/login", map[string]string{"email": name + "@example.invalid", "password": password}, 200)
		cookies[name] = response.Cookies()[0]
		var login struct {
			Redirect string `json:"redirect"`
		}
		if err := json.Unmarshal(body, &login); err != nil {
			t.Fatal(err)
		}
		expected, _ := models.DashboardPath(models.Role(role))
		if login.Redirect != expected {
			t.Fatal("incorrect role login redirect")
		}
		response, _ = request(name, "GET", "/dashboard", nil, 303)
		if response.Header.Get("Location") != expected {
			t.Fatal("incorrect dashboard entry redirect")
		}
		for _, target := range []string{"customer", "vendor", "rider", "admin"} {
			path, _ := models.DashboardPath(models.Role(target))
			want := 403
			if role == target {
				want = 200
			}
			request(name, "GET", path, nil, want)
		}
	}
	endpoints := map[string]string{"customer": "/api/customer/orders", "vendor": "/api/vendor/products", "rider": "/api/rider/deliveries", "admin": "/api/admin/users"}
	for name, role := range roles {
		for required, path := range endpoints {
			want := 403
			if role == required {
				want = 200
			}
			request(name, "GET", path, nil, want)
		}
	}
	for _, path := range endpoints {
		request("", "GET", path, nil, 401)
	}
	for _, pair := range []struct {
		base       string
		own, other int64
	}{{"orders", o1, o2}, {"addresses", a1, a2}, {"payments", payment1, payment2}} {
		request("c1", "GET", fmt.Sprintf("/api/customer/%s/%d", pair.base, pair.own), nil, 200)
		_, foreign := request("c1", "GET", fmt.Sprintf("/api/customer/%s/%d", pair.base, pair.other), nil, 403)
		_, missing := request("c1", "GET", fmt.Sprintf("/api/customer/%s/999999999", pair.base), nil, 403)
		if !bytes.Equal(foreign, missing) {
			t.Fatal("ownership denial disclosed resource existence")
		}
	}
	_, body := request("c1", "GET", fmt.Sprintf("/api/customer/orders?customer_id=%d", users["c2"]), nil, 200)
	var listed struct {
		Orders []struct {
			ID int64 `json:"id"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(body, &listed); err != nil || len(listed.Orders) != 1 || listed.Orders[0].ID != o1 {
		t.Fatal("customer order list escaped ownership filter")
	}
	request("v1", "GET", fmt.Sprintf("/api/vendor/products/%d", p1), nil, 200)
	request("v1", "GET", fmt.Sprintf("/api/vendor/products/%d", p2), nil, 403)
	request("v1", "PATCH", fmt.Sprintf("/api/vendor/products/%d", p2), map[string]string{"name": "Stolen product", "price": "1.00"}, 403)
	request("v1", "DELETE", fmt.Sprintf("/api/vendor/products/%d", p2), nil, 403)
	request("v1", "GET", fmt.Sprintf("/api/vendor/orders/%d", o2), nil, 403)
	request("v1", "PATCH", "/api/vendor/shop", map[string]any{"name": "Hijack", "address": "Street", "user_id": users["v2"]}, 400)
	request("v1", "PATCH", "/api/vendor/shop", map[string]string{"name": "Changed shop", "address": "Updated address"}, 200)
	var unchanged string
	if err := pool.QueryRow(ctx, "SELECT name FROM vendors WHERE id=$1", v2).Scan(&unchanged); err != nil || unchanged != "Shop two" {
		t.Fatal("shop update changed another owner")
	}
	request("v1", "POST", "/api/vendor/products", map[string]any{"name": "Hijack", "price": "2.00", "vendor_id": v2}, 400)
	_, body = request("v1", "POST", "/api/vendor/products", map[string]string{"name": "Owned new product", "price": "123.45"}, 200)
	var created struct {
		Product struct {
			ID       int64 `json:"id"`
			VendorID int64 `json:"vendor_id"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if created.Product.VendorID != v1 {
		t.Fatal("product creation trusted an unowned vendor")
	}
	request("v1", "PATCH", fmt.Sprintf("/api/vendor/products/%d", created.Product.ID), map[string]string{"name": "Owned edited product", "price": "125.50"}, 200)
	request("v1", "DELETE", fmt.Sprintf("/api/vendor/products/%d", created.Product.ID), nil, 204)
	request("v1", "GET", fmt.Sprintf("/api/vendor/products/%d", created.Product.ID), nil, 403)
	request("r1", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d1), nil, 200)
	request("r1", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d2), nil, 403)
	request("r1", "GET", fmt.Sprintf("/api/rider/deliveries/%d", unassigned), nil, 403)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d2), map[string]string{"status": "picked_up"}, 403)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d2), map[string]any{"status": "picked_up", "rider_id": users["r1"]}, 400)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d1), map[string]string{"status": "delivered"}, 403)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d1), map[string]string{"status": "picked_up"}, 200)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d1), map[string]string{"status": "delivered"}, 200)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", d1), map[string]string{"status": "picked_up"}, 403)
	if _, err := pool.Exec(ctx, "UPDATE deliveries SET rider_id=$1 WHERE id=$2", users["r1"], d2); err != nil {
		t.Fatal(err)
	}
	request("r2", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d2), nil, 403)
	request("r1", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d2), nil, 200)
	request("v1", "PATCH", fmt.Sprintf("/api/admin/vendors/%d/status", v1), map[string]string{"status": "approved"}, 403)
	request("admin", "PATCH", fmt.Sprintf("/api/admin/vendors/%d/status", v1), map[string]string{"status": "approved"}, 422)
	request("c1", "POST", "/api/auth/register", map[string]string{"name": "Privilege attempt", "email": "escalation@example.invalid", "phone": "+254712345678", "password": password, "role": "admin"}, 400)
	request("c1", "POST", "/api/auth/register", map[string]string{"name": "Public user", "email": "public@example.invalid", "phone": "+254712345678", "password": password}, 201)
	var registeredRole string
	if err := pool.QueryRow(ctx, "SELECT role FROM users WHERE email='public@example.invalid'").Scan(&registeredRole); err != nil || registeredRole != "customer" {
		t.Fatal("public user gained a privileged role")
	}
	request("v1", "PATCH", "/api/vendor/shop", map[string]string{"name": "Escalation", "address": "Street", "role": "admin", "status": "approved"}, 400)
	if _, err := pool.Exec(ctx, "UPDATE users SET role='customer' WHERE id=$1", users["admin"]); err != nil {
		t.Fatal(err)
	}
	request("admin", "GET", "/api/admin/users", nil, 403)
	response, _ := request("admin", "GET", "/dashboard", nil, 303)
	if response.Header.Get("Location") != "/customer/dashboard" {
		t.Fatal("role change did not affect existing session")
	}
}
