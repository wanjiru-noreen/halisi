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

type captureMPesa struct {
	calls        int
	callbackURL  string
	err          error
	authorizeErr error
}

func (m *captureMPesa) Authorize(_ context.Context) (string, error) {
	return "test-token", m.authorizeErr
}

func (m *captureMPesa) Initiate(_ context.Context, _ string, input services.STKRequest) (services.STKResponse, error) {
	m.calls++
	m.callbackURL = input.CallbackURL
	if m.err != nil {
		return services.STKResponse{}, m.err
	}
	return services.STKResponse{
		MerchantRequestID: fmt.Sprintf("merchant-%d", m.calls),
		CheckoutRequestID: fmt.Sprintf("checkout-%d", m.calls),
		ResponseCode:      "0",
	}, nil
}

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
	if err := m.Steps(6); err != nil {
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
	// These fixtures represent already-approved delivery partners, not new applicants.
	for _, name := range []string{"r1", "r2"} {
		if _, err := pool.Exec(ctx, "INSERT INTO rider_profiles(user_id,full_name,phone,city,status) VALUES($1,$2,'+254712345678','Nairobi','approved')", users[name], name); err != nil {
			t.Fatal(err)
		}
	}
	v1 := id("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Shop one','Nairobi','approved',TRUE) RETURNING id", users["v1"])
	v2 := id("INSERT INTO vendors(user_id,name,address,status,is_active) VALUES($1,'Shop two','Mombasa','approved',TRUE) RETURNING id", users["v2"])
	p1 := id("INSERT INTO products(vendor_id,name,price) VALUES($1,'Cylinder one',1000) RETURNING id", v1)
	p2 := id("INSERT INTO products(vendor_id,name,price) VALUES($1,'Cylinder two',2000) RETURNING id", v2)
	a1 := id("INSERT INTO addresses(user_id,address_line,city) VALUES($1,'Private one','Nairobi') RETURNING id", users["c1"])
	a2 := id("INSERT INTO addresses(user_id,address_line,city) VALUES($1,'Private two','Mombasa') RETURNING id", users["c2"])
	o1 := id("INSERT INTO orders(customer_id,vendor_id,status,total,delivery_address) VALUES($1,$2,'confirmed',1000,'Private one') RETURNING id", users["c1"], v1)
	o2 := id("INSERT INTO orders(customer_id,vendor_id,status,total,delivery_address) VALUES($1,$2,'confirmed',2000,'Private two') RETURNING id", users["c2"], v2)
	payment1 := id("INSERT INTO payments(order_id,amount) VALUES($1,1000) RETURNING id", o1)
	payment2 := id("INSERT INTO payments(order_id,amount) VALUES($1,2000) RETURNING id", o2)
	d1 := id("INSERT INTO deliveries(order_id,rider_id,status) VALUES($1,$2,'assigned') RETURNING id", o1, users["r1"])
	d2 := id("INSERT INTO deliveries(order_id,rider_id,status) VALUES($1,$2,'assigned') RETURNING id", o2, users["r2"])
	o3 := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,100,'Unassigned') RETURNING id", users["c2"], v2)
	unassigned := id("INSERT INTO deliveries(order_id) VALUES($1) RETURNING id", o3)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	service, err := services.NewAuth(pool, services.WithVerification(&captureMailer{}, "https://halisi.example"))
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(service, "production")
	resources := handlers.NewResources(pool, auth)
	checkout := handlers.NewCheckout(pool, auth)
	fakeMPesa := &captureMPesa{}
	mpesa := handlers.NewMPesa(pool, auth, fakeMPesa, "https://halisi.example", "174379", "test-passkey", "CustomerPayBillOnline")
	server := httptest.NewTLSServer(routes.New(pool, auth.Routes, resources.Routes, checkout.Routes, mpesa.Routes))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	cookies := map[string]*http.Cookie{}
	requestWithHeaders := func(actor, method, path string, body any, headers map[string]string, want int) (*http.Response, []byte) {
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
		for name, value := range headers {
			r.Header.Set(name, value)
		}
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
	request := func(actor, method, path string, body any, want int) (*http.Response, []byte) {
		return requestWithHeaders(actor, method, path, body, nil, want)
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
	_, body = request("v1", "POST", "/api/vendor/products", map[string]any{"name": "Owned new product", "price": "123.45", "quantity": 4}, 200)
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
	var initialQuantity int32
	if err := pool.QueryRow(ctx, "SELECT quantity FROM vendor_inventory WHERE product_id=$1", created.Product.ID).Scan(&initialQuantity); err != nil || initialQuantity != 4 {
		t.Fatal("product creation did not initialize inventory")
	}
	request("v1", "PATCH", fmt.Sprintf("/api/vendor/products/%d", created.Product.ID), map[string]any{"name": "Owned edited product", "price": "125.50", "quantity": 6}, 200)
	if err := pool.QueryRow(ctx, "SELECT quantity FROM vendor_inventory WHERE product_id=$1", created.Product.ID).Scan(&initialQuantity); err != nil || initialQuantity != 6 {
		t.Fatal("product update did not update inventory")
	}
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
	if _, err := pool.Exec(ctx, `INSERT INTO vendor_inventory(product_id,vendor_id,quantity,reserved_quantity) VALUES($1,$2,10,2)`, p1, v1); err != nil {
		t.Fatal(err)
	}
	// Restore the reviewed commerce fixture after the owner-edit isolation test.
	// The onboarding integration test covers the real administrator approval flow.
	if _, err := pool.Exec(ctx, "UPDATE vendors SET status='approved',is_active=TRUE WHERE id=$1", v1); err != nil {
		t.Fatal(err)
	}
	failedOrder := id("INSERT INTO orders(customer_id,vendor_id,address_id,status,total,delivery_address) VALUES($1,$2,$3,'confirmed',2000,'Failure test') RETURNING id", users["c1"], v1, a1)
	if _, err := pool.Exec(ctx, `INSERT INTO order_items(order_id,vendor_id,product_id,quantity,unit_price) VALUES($1,$2,$3,2,1000)`, failedOrder, v1, p1); err != nil {
		t.Fatal(err)
	}
	failedDelivery := id("UPDATE deliveries SET rider_id=$2,status='assigned' WHERE order_id=$1 RETURNING id", failedOrder, users["r1"])
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", failedDelivery), map[string]string{"status": "failed"}, 200)
	var reserved int32
	if err := pool.QueryRow(ctx, "SELECT reserved_quantity FROM vendor_inventory WHERE product_id=$1", p1).Scan(&reserved); err != nil || reserved != 0 {
		t.Fatal("failed delivery did not release reserved stock")
	}
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", failedDelivery), map[string]string{"status": "failed"}, 403)
	if err := pool.QueryRow(ctx, "SELECT reserved_quantity FROM vendor_inventory WHERE product_id=$1", p1).Scan(&reserved); err != nil || reserved != 0 {
		t.Fatal("repeated failed transition changed reserved stock")
	}
	if _, err := pool.Exec(ctx, "UPDATE deliveries SET rider_id=$1 WHERE id=$2", users["r1"], d2); err != nil {
		t.Fatal(err)
	}
	request("r2", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d2), nil, 403)
	request("r1", "GET", fmt.Sprintf("/api/rider/deliveries/%d", d2), nil, 200)
	request("v1", "PATCH", fmt.Sprintf("/api/admin/vendors/%d/status", v1), map[string]string{"status": "approved"}, 403)
	request("admin", "PATCH", fmt.Sprintf("/api/admin/vendors/%d/status", v1), map[string]string{"status": "approved"}, 422)
	if _, err := pool.Exec(ctx, "UPDATE vendors SET status='approved',is_active=TRUE WHERE id=$1", v1); err != nil {
		t.Fatal(err)
	}
	_, body = request("c1", "POST", "/api/customer/checkout", map[string]any{
		"address_id": a1,
		"items":      []map[string]any{{"product_id": p1, "quantity": 2}},
	}, 201)
	var checkedOut struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(body, &checkedOut); err != nil {
		t.Fatal(err)
	}
	var pendingDeliveries int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE order_id=$1", checkedOut.Order.ID).Scan(&pendingDeliveries); err != nil || pendingDeliveries != 0 {
		t.Fatal("unconfirmed checkout created a delivery")
	}
	request("v1", "POST", fmt.Sprintf("/api/vendor/orders/%d/confirm", checkedOut.Order.ID), nil, 200)
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE order_id=$1 AND status='pending'", checkedOut.Order.ID).Scan(&pendingDeliveries); err != nil || pendingDeliveries != 1 {
		t.Fatal("confirmation did not create exactly one pending delivery")
	}
	if err := pool.QueryRow(ctx, "SELECT reserved_quantity FROM vendor_inventory WHERE product_id=$1", p1).Scan(&reserved); err != nil || reserved != 2 {
		t.Fatal("checkout did not reserve inventory")
	}
	paymentPath := fmt.Sprintf("/api/customer/orders/%d/payments/mpesa", checkedOut.Order.ID)
	failurePaymentPath := fmt.Sprintf("/api/customer/orders/%d/payments/mpesa", failedOrder)
	paymentBody := map[string]string{"phone_number": "+254 712 345 678"}
	paymentHeaders := map[string]string{"Idempotency-Key": "checkout-payment-1"}
	requestWithHeaders("c2", "POST", paymentPath, paymentBody, map[string]string{"Idempotency-Key": "foreign-payment"}, 404)
	_, body = requestWithHeaders("c1", "POST", paymentPath, paymentBody, paymentHeaders, 202)
	var initiated struct {
		Payment struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(body, &initiated); err != nil || initiated.Payment.Status != "processing" || fakeMPesa.calls != 1 {
		t.Fatal("M-Pesa payment was not started")
	}
	requestWithHeaders("c1", "POST", paymentPath, paymentBody, paymentHeaders, 202)
	requestWithHeaders("c1", "POST", paymentPath, map[string]string{"phone_number": "0112345678"}, paymentHeaders, 409)
	requestWithHeaders("c1", "POST", paymentPath, paymentBody, map[string]string{"Idempotency-Key": "alternate-key"}, 202)
	requestWithHeaders("c1", "POST", failurePaymentPath, paymentBody, paymentHeaders, 409)
	if fakeMPesa.calls != 1 {
		t.Fatal("duplicate payment requests initiated another STK prompt")
	}
	ratePath := fmt.Sprintf("/api/admin/vendors/%d/commission-rate", v1)
	requestWithHeaders("admin", "PATCH", ratePath, map[string]int{"rate_bps": 1250}, nil, 200)
	callbackToken := strings.TrimPrefix(fakeMPesa.callbackURL, "https://halisi.example/api/mpesa/callback/")
	callback := map[string]any{"Body": map[string]any{"stkCallback": map[string]any{
		"MerchantRequestID": "merchant-1", "CheckoutRequestID": "checkout-1", "ResultCode": 0, "ResultDesc": "Success",
		"CallbackMetadata": map[string]any{"Item": []map[string]any{
			{"Name": "Amount", "Value": 2000}, {"Name": "MpesaReceiptNumber", "Value": "ABC123"}, {"Name": "PhoneNumber", "Value": 254712345678},
		}},
	}}}
	request("", "POST", "/api/mpesa/callback/wrong-token", callback, 404)
	wrongAmountCallback := map[string]any{"Body": map[string]any{"stkCallback": map[string]any{
		"MerchantRequestID": "merchant-1", "CheckoutRequestID": "checkout-1", "ResultCode": 0, "ResultDesc": "Success",
		"CallbackMetadata": map[string]any{"Item": []map[string]any{
			{"Name": "Amount", "Value": 2001}, {"Name": "MpesaReceiptNumber", "Value": "BAD123"}, {"Name": "PhoneNumber", "Value": 254712345678},
		}},
	}}}
	request("", "POST", "/api/mpesa/callback/"+callbackToken, wrongAmountCallback, 400)
	request("", "POST", "/api/mpesa/callback/"+callbackToken, callback, 200)
	request("", "POST", "/api/mpesa/callback/"+callbackToken, callback, 200)
	var paymentStatus, orderStatus, receipt string
	if err := pool.QueryRow(ctx, "SELECT status,reference FROM payments WHERE id=$1", initiated.Payment.ID).Scan(&paymentStatus, &receipt); err != nil || paymentStatus != "completed" || receipt != "ABC123" {
		t.Fatal("successful M-Pesa callback did not complete the payment")
	}
	if err := pool.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", checkedOut.Order.ID).Scan(&orderStatus); err != nil || orderStatus != "confirmed" {
		t.Fatal("successful M-Pesa callback did not confirm the order")
	}
	var gross, fee, vendorShare string
	var commissionRate int32
	if err := pool.QueryRow(ctx, "SELECT gross_amount::text,commission_rate_bps,commission_amount::text,vendor_amount::text FROM order_commissions WHERE order_id=$1", checkedOut.Order.ID).Scan(&gross, &commissionRate, &fee, &vendorShare); err != nil || gross != "2000.00" || commissionRate != 1000 || fee != "200.00" || vendorShare != "1800.00" {
		t.Fatal("successful payment did not create an auditable commission entry")
	}
	requestWithHeaders("v1", "PATCH", ratePath, map[string]int{"rate_bps": 1250}, nil, 403)
	requestWithHeaders("admin", "PATCH", ratePath, map[string]int{"rate_bps": 10001}, nil, 422)
	requestWithHeaders("admin", "PATCH", ratePath, map[string]int{"rate_bps": 1500}, nil, 200)
	if err := pool.QueryRow(ctx, "SELECT commission_rate_bps FROM order_commissions WHERE order_id=$1", checkedOut.Order.ID).Scan(&commissionRate); err != nil || commissionRate != 1000 {
		t.Fatal("commission rate change rewrote the historical commission snapshot")
	}
	deliveredID := id("UPDATE deliveries SET status='assigned',rider_id=$2 WHERE order_id=$1 RETURNING id", checkedOut.Order.ID, users["r1"])
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", deliveredID), map[string]string{"status": "picked_up"}, 200)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", deliveredID), map[string]string{"status": "delivered"}, 200)
	if err := pool.QueryRow(ctx, "SELECT status FROM deliveries WHERE id=$1", deliveredID).Scan(&orderStatus); err != nil || orderStatus != "delivered" {
		t.Fatal("test delivery did not reach its delivered state", err)
	}
	zeroShareOrder := id("INSERT INTO orders(customer_id,vendor_id,address_id,status,total,delivery_address) VALUES($1,$2,$3,'confirmed',100,'Zero share') RETURNING id", users["c1"], v1, a1)
	zeroSharePayment := id("INSERT INTO payments(order_id,amount,status) VALUES($1,100,'completed') RETURNING id", zeroShareOrder)
	if _, err := pool.Exec(ctx, `INSERT INTO order_commissions(order_id,payment_id,vendor_id,gross_amount,commission_rate_bps,commission_amount,vendor_amount,currency) VALUES($1,$2,$3,100,10000,100,0,'KES')`, zeroShareOrder, zeroSharePayment, v1); err != nil {
		t.Fatal(err)
	}
	zeroShareDelivery := id("INSERT INTO deliveries(order_id) VALUES($1) RETURNING id", zeroShareOrder)
	if _, err := pool.Exec(ctx, "UPDATE deliveries SET status='assigned',rider_id=$2 WHERE id=$1", zeroShareDelivery, users["r1"]); err != nil {
		t.Fatal(err)
	}
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", zeroShareDelivery), map[string]string{"status": "picked_up"}, 200)
	request("r1", "PATCH", fmt.Sprintf("/api/rider/deliveries/%d", zeroShareDelivery), map[string]string{"status": "delivered"}, 200)
	settlementPath := fmt.Sprintf("/api/admin/vendors/%d/settlements", v1)
	requestWithHeaders("c1", "POST", settlementPath, nil, nil, 403)
	_, body = requestWithHeaders("admin", "POST", settlementPath, nil, nil, 200)
	var createdSettlement struct {
		Settlement struct {
			ID     int64  `json:"id"`
			Amount string `json:"amount"`
			Status string `json:"status"`
		} `json:"settlement"`
	}
	if err := json.Unmarshal(body, &createdSettlement); err != nil || createdSettlement.Settlement.Amount != "1800.00" || createdSettlement.Settlement.Status != "pending" {
		t.Fatal("settlement did not capture the eligible vendor balance")
	}
	requestWithHeaders("admin", "POST", settlementPath, nil, nil, 409)
	settlementUpdatePath := fmt.Sprintf("/api/admin/settlements/%d", createdSettlement.Settlement.ID)
	requestWithHeaders("admin", "PATCH", settlementUpdatePath, map[string]string{"status": "failed", "failure_reason": "Bank transfer returned an error"}, nil, 200)
	_, body = requestWithHeaders("admin", "POST", settlementPath, nil, nil, 200)
	if err := json.Unmarshal(body, &createdSettlement); err != nil || createdSettlement.Settlement.Amount != "1800.00" {
		t.Fatal("failed settlement did not release commissions for retry")
	}
	settlementUpdatePath = fmt.Sprintf("/api/admin/settlements/%d", createdSettlement.Settlement.ID)
	requestWithHeaders("admin", "PATCH", settlementUpdatePath, map[string]string{"status": "paid", "payout_reference": "BATCH-2026-001"}, nil, 200)
	requestWithHeaders("admin", "PATCH", settlementUpdatePath, map[string]string{"status": "paid", "payout_reference": "BATCH-2026-001"}, nil, 409)
	_, body = requestWithHeaders("admin", "GET", "/api/admin/settlements", nil, nil, 200)
	if !bytes.Contains(body, []byte(`"payout_reference":"BATCH-2026-001"`)) || !bytes.Contains(body, []byte(`"failure_reason":"Bank transfer returned an error"`)) || bytes.Count(body, []byte(`"commission_id"`)) < 2 {
		t.Fatal("administrator settlement history omitted payout, failed-attempt or item audit details")
	}
	_, body = requestWithHeaders("v1", "GET", "/api/vendor/commissions", nil, nil, 200)
	if !bytes.Contains(body, []byte(`"settlement_status":"paid"`)) || !bytes.Contains(body, []byte(`"settlement_status":"not_payable"`)) {
		t.Fatal("vendor commission ledger did not show paid and zero-share states")
	}
	_, body = request("c1", "GET", fmt.Sprintf("/api/customer/payments/%d", initiated.Payment.ID), nil, 200)
	if bytes.Contains(body, []byte("callback_token_hash")) || bytes.Contains(body, []byte("request_key")) || bytes.Contains(body, []byte("phone_number")) {
		t.Fatal("customer payment response exposed internal M-Pesa metadata")
	}
	_, body = requestWithHeaders("c1", "POST", failurePaymentPath, paymentBody, map[string]string{"Idempotency-Key": "failed-payment-1"}, 202)
	if err := json.Unmarshal(body, &initiated); err != nil {
		t.Fatal(err)
	}
	failureToken := strings.TrimPrefix(fakeMPesa.callbackURL, "https://halisi.example/api/mpesa/callback/")
	failureCallback := map[string]any{"Body": map[string]any{"stkCallback": map[string]any{
		"MerchantRequestID": "merchant-2", "CheckoutRequestID": "checkout-2", "ResultCode": 1032, "ResultDesc": "Cancelled by user",
	}}}
	request("", "POST", "/api/mpesa/callback/"+failureToken, failureCallback, 200)
	request("", "POST", "/api/mpesa/callback/"+failureToken, failureCallback, 200)
	if err := pool.QueryRow(ctx, "SELECT status FROM payments WHERE id=$1", initiated.Payment.ID).Scan(&paymentStatus); err != nil || paymentStatus != "failed" {
		t.Fatal("failed M-Pesa callback did not fail the payment")
	}
	fakeMPesa.authorizeErr = services.ErrSTKNotSubmitted
	_, body = requestWithHeaders("c1", "POST", failurePaymentPath, paymentBody, map[string]string{"Idempotency-Key": "preflight-payment-1"}, 502)
	if err := json.Unmarshal(body, &initiated); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT status FROM payments WHERE id=$1", initiated.Payment.ID).Scan(&paymentStatus); err != nil || paymentStatus != "failed" {
		t.Fatal("pre-STK failure left payment attempt active")
	}
	fakeMPesa.authorizeErr = nil
	requestWithHeaders("c1", "POST", failurePaymentPath, paymentBody, map[string]string{"Idempotency-Key": "preflight-payment-retry"}, 202)
	if fakeMPesa.calls != 3 {
		t.Fatal("customer could not retry after pre-STK failure")
	}
	recoveryOrder := id("INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,1000,'Recovery test') RETURNING id", users["c1"], v1)
	if _, err := pool.Exec(ctx, `INSERT INTO payments(order_id,amount,status,provider,request_key,phone_number,callback_token_hash) VALUES($1,1000,'pending','mpesa','stranded-attempt','254712345678',repeat('a',64))`, recoveryOrder); err != nil {
		t.Fatal(err)
	}
	recoveryPath := fmt.Sprintf("/api/customer/orders/%d/payments/mpesa", recoveryOrder)
	requestWithHeaders("c1", "POST", recoveryPath, paymentBody, map[string]string{"Idempotency-Key": "resume-stranded-attempt"}, 202)
	if fakeMPesa.calls != 4 {
		t.Fatal("pending payment attempt was not resumed")
	}
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
