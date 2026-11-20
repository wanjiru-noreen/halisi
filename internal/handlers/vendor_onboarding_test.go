package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"github.com/wanjiru-noreen/halisi/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func TestVendorOnboardingPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL for vendor onboarding integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	adminDB, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("cannot initialize test database")
	}
	defer adminDB.Close()
	schema := fmt.Sprintf("vendors_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := adminDB.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("cleanup failed")
		}
	}()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid test database URL")
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	m, err := database.NewMigrator(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}
	pool, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store, err := storage.OpenDocuments(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := &services.VendorOnboarding{Pool: pool, Files: store}
	authService, err := services.NewAuth(pool)
	if err != nil {
		t.Fatal(err)
	}
	auth := handlers.NewAuth(authService, "production")
	onboarding := handlers.NewVendorOnboarding(auth, service)
	resources := handlers.NewResources(pool, auth, service)
	storefront := handlers.NewStorefront(pool)
	server := httptest.NewTLSServer(routes.New(pool, auth.Routes, resources.Routes, onboarding.Routes, storefront.Routes))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	const password = "vendor workflow test passphrase"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), services.PasswordCost)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	cookies := map[string]*http.Cookie{}
	for name, role := range map[string]string{"owner": "customer", "customer": "customer", "other": "vendor", "admin": "admin", "unverified": "customer"} {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(name,email,password_hash,role,email_verified_at) VALUES($1,$2,$3,$4,CASE WHEN $1='unverified' THEN NULL ELSE NOW() END) RETURNING id", name, name+"@example.invalid", string(hash), role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[name] = id
		session, err := authService.Login(ctx, name+"@example.invalid", password, "")
		if err != nil {
			t.Fatal(err)
		}
		cookies[name] = &http.Cookie{Name: "__Host-halisi_session", Value: session.Token}
	}
	send := func(actor, method, path, contentType string, reader io.Reader, want int) (*http.Response, []byte) {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		if cookies[actor] != nil {
			r.AddCookie(cookies[actor])
		}
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s %s got %d, want %d: %s", actor, method, path, response.StatusCode, want, body)
		}
		return response, body
	}
	request := func(actor, method, path string, input any, want int) (*http.Response, []byte) {
		t.Helper()
		var reader io.Reader
		typ := ""
		if input != nil {
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			reader = bytes.NewReader(body)
			typ = "application/json"
		}
		return send(actor, method, path, typ, reader, want)
	}
	hours := map[string]services.Hours{}
	for _, d := range []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"} {
		hours[d] = services.Hours{Open: "08:00", Close: "18:00"}
	}
	profile := services.VendorInput{Name: "Nairobi LPG", ContactEmail: "shop@example.invalid", ContactPhone: "+254712345678", Address: "Kenyatta Avenue", City: "Nairobi", OperatingHours: hours}
	request("", "PUT", "/api/vendor/application", profile, 401)
	request("unverified", "PUT", "/api/vendor/application", profile, 403)
	request("admin", "PUT", "/api/vendor/application", profile, 403)
	_, body := request("owner", "PUT", "/api/vendor/application", profile, 200)
	var result struct {
		Application struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			Active bool   `json:"is_active"`
		} `json:"application"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	vendorID := result.Application.ID
	if vendorID == 0 || result.Application.Status != "pending" || result.Application.Active {
		t.Fatal("new vendor was not pending/inactive")
	}
	var role string
	if err := pool.QueryRow(ctx, "SELECT role FROM users WHERE id=$1", ids["owner"]).Scan(&role); err != nil || role != "vendor" {
		t.Fatal("verified customer did not become a vendor")
	}
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 422)
	reviewPath := fmt.Sprintf("/api/admin/vendor-applications/%d/review", vendorID)
	decision := func(status string) map[string]any {
		_, body := request("admin", "GET", fmt.Sprintf("/api/admin/vendor-applications/%d", vendorID), nil, 200)
		var detail struct {
			Application struct {
				ReviewVersion int64 `json:"review_version"`
			} `json:"application"`
		}
		if err := json.Unmarshal(body, &detail); err != nil {
			t.Fatal(err)
		}
		return map[string]any{"status": status, "reason": "Reviewed business information and documents", "review_version": detail.Application.ReviewVersion}
	}
	request("owner", "POST", reviewPath, decision("approved"), 403)
	request("admin", "POST", reviewPath, decision("approved"), 403)
	pdf := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\n%%EOF\n")
	upload := func(actor, kind, name, contentType string, data []byte, want int) int64 {
		t.Helper()
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		headers := textproto.MIMEHeader{}
		headers.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "document", "filename": name}))
		headers.Set("Content-Type", contentType)
		part, err := writer.CreatePart(headers)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		_, body := send(actor, "POST", "/api/vendor/documents/"+kind, writer.FormDataContentType(), &buffer, want)
		if strings.Contains(string(body), "storage_key") {
			t.Fatal("private storage key exposed")
		}
		if want != 201 {
			return 0
		}
		var result struct {
			Document struct {
				ID int64 `json:"id"`
			} `json:"document"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		return result.Document.ID
	}
	upload("owner", "business_registration", "evil.html", "text/html", []byte("<script>bad</script>"), 400)
	upload("owner", "business_registration", "huge.pdf", "application/pdf", bytes.Repeat([]byte("x"), storage.MaxDocumentSize+1), 400)
	documentID := upload("owner", "business_registration", "registration.pdf", "application/pdf", pdf, 201)
	licenceID := upload("owner", "lpg_license", "licence.pdf", "application/pdf", pdf, 201)
	documentPath := fmt.Sprintf("/api/vendor/documents/%d", documentID)
	response, download := request("owner", "GET", documentPath, nil, 200)
	if !bytes.Equal(download, pdf) || !strings.HasPrefix(response.Header.Get("Content-Disposition"), "attachment") || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("private document download was unsafe")
	}
	request("admin", "GET", documentPath, nil, 200)
	request("other", "GET", documentPath, nil, 403)
	request("customer", "GET", documentPath, nil, 403)
	request("", "GET", documentPath, nil, 401)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	assertSearch := func(expected int) {
		t.Helper()
		_, body := request("customer", "GET", "/api/customer/vendors?q=Nairobi", nil, 200)
		var search struct {
			Vendors []any `json:"vendors"`
		}
		if err := json.Unmarshal(body, &search); err != nil || len(search.Vendors) != expected {
			t.Fatalf("search visibility mismatch: %s", body)
		}
		_, body = request("", "GET", "/api/storefront/vendors?q=Nairobi", nil, 200)
		if err := json.Unmarshal(body, &search); err != nil || len(search.Vendors) != expected {
			t.Fatalf("public storefront visibility mismatch: %s", body)
		}
	}
	assertSale := func(allowed bool) {
		t.Helper()
		_, err := pool.Exec(ctx, "INSERT INTO orders(customer_id,vendor_id,total,delivery_address) VALUES($1,$2,1000,'Nairobi')", ids["customer"], vendorID)
		if allowed {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("unverified sale was not blocked: %v", err)
			}
		}
	}
	assertSearch(0)
	assertSale(false)
	request("admin", "POST", reviewPath, map[string]string{"status": "approved", "reason": "Missing submission version"}, 422)
	stale := decision("approved")
	upload("owner", "lpg_license", "replacement-after-review.pdf", "application/pdf", pdf, 201)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	request("admin", "POST", reviewPath, stale, 409)
	request("admin", "PATCH", fmt.Sprintf("/api/admin/vendors/%d/status", vendorID), stale, 409)
	assertSearch(0)
	assertSale(false)
	var prematureReviews int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM vendor_review_events WHERE vendor_id=$1", vendorID).Scan(&prematureReviews); err != nil || prematureReviews != 0 {
		t.Fatal("stale decision created a review event")
	}
	request("admin", "POST", reviewPath, decision("rejected"), 200)
	_, body = request("owner", "GET", "/api/vendor/application", nil, 200)
	if !strings.Contains(string(body), "rejected") {
		t.Fatal("rejection status was not visible")
	}
	profile.Address = "Corrected Kenyatta Avenue"
	request("owner", "PUT", "/api/vendor/application", profile, 200)
	var oldKey string
	if err := pool.QueryRow(ctx, "SELECT storage_key FROM vendor_documents WHERE id=$1", licenceID).Scan(&oldKey); err != nil {
		t.Fatal(err)
	}
	upload("owner", "lpg_license", "corrected.pdf", "application/pdf", pdf, 201)
	if f, err := store.Open(oldKey); err == nil {
		f.Close()
		t.Fatal("replaced private document was retained")
	}
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	stale = decision("approved")
	profile.Address = "Changed after the administrator opened the application"
	request("owner", "PUT", "/api/vendor/application", profile, 200)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	request("admin", "POST", reviewPath, stale, 409)
	if _, err := pool.Exec(ctx, "UPDATE users SET role='admin' WHERE id=$1", ids["owner"]); err != nil {
		t.Fatal(err)
	}
	request("owner", "POST", reviewPath, decision("approved"), 403)
	if _, err := pool.Exec(ctx, "UPDATE users SET role='vendor' WHERE id=$1", ids["owner"]); err != nil {
		t.Fatal(err)
	}
	request("admin", "POST", reviewPath, decision("approved"), 200)
	assertSearch(1)
	assertSale(true)
	request("owner", "PATCH", "/api/vendor/shop", map[string]string{"name": "Updated Nairobi LPG", "address": "New Nairobi address"}, 200)
	assertSearch(0)
	assertSale(false)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	request("admin", "POST", reviewPath, decision("approved"), 200)
	request("admin", "POST", reviewPath, decision("suspended"), 200)
	assertSearch(0)
	assertSale(false)
	request("owner", "PUT", "/api/vendor/application", profile, 403)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 403)
	upload("owner", "lpg_license", "blocked.pdf", "application/pdf", pdf, 403)
	request("admin", "POST", reviewPath, decision("pending"), 200)
	request("owner", "POST", "/api/vendor/application/submit", map[string]string{}, 200)
	request("admin", "POST", reviewPath, decision("approved"), 200)
	assertSearch(1)
	var reviews int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM vendor_review_events WHERE vendor_id=$1", vendorID).Scan(&reviews); err != nil || reviews != 6 {
		t.Fatalf("review history not recorded: %d %v", reviews, err)
	}
	request("owner", "GET", "/vendor/register", nil, 200)
	request("owner", "GET", "/vendor/verification", nil, 200)
	request("admin", "GET", "/admin/vendor-verification", nil, 200)
	request("owner", "GET", "/admin/vendor-verification", nil, 403)
	// More than 100 vendors must remain reachable through the review list.
	if _, err := pool.Exec(ctx, `WITH owners AS (
INSERT INTO users(name,email,password_hash,role)
SELECT 'Review fixture '||g,'review-fixture-'||g||'@example.invalid','test-only-hash','vendor' FROM generate_series(1,120) g RETURNING id
) INSERT INTO vendors(user_id,name,address) SELECT id,'Later application '||id,'Nairobi' FROM owners`); err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	after := int64(0)
	pages := 0
	for {
		_, body := request("admin", "GET", fmt.Sprintf("/api/admin/vendor-applications?after_id=%d", after), nil, 200)
		var page struct {
			Applications []struct {
				ID int64 `json:"id"`
			} `json:"applications"`
			Next int64 `json:"next_after_id"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Applications) > 50 {
			t.Fatal("review page exceeded limit")
		}
		for _, v := range page.Applications {
			if seen[v.ID] {
				t.Fatal("duplicate application across pages")
			}
			seen[v.ID] = true
		}
		pages++
		if pages > 4 {
			t.Fatal("pagination did not terminate")
		}
		if page.Next == 0 {
			break
		}
		if page.Next <= after {
			t.Fatal("pagination cursor did not advance")
		}
		after = page.Next
	}
	if len(seen) != 121 || pages != 3 {
		t.Fatalf("later applications unreachable: %d applications, %d pages", len(seen), pages)
	}
	var lastID int64
	if err := pool.QueryRow(ctx, "SELECT max(id) FROM vendors").Scan(&lastID); err != nil {
		t.Fatal(err)
	}
	if !seen[lastID] {
		t.Fatal("latest application missing")
	}
	request("admin", "GET", fmt.Sprintf("/api/admin/vendor-applications/%d", lastID), nil, 200)
	request("customer", "GET", "/api/admin/vendor-applications", nil, 403)
	request("", "GET", "/api/admin/vendor-applications", nil, 401)
	request("admin", "GET", "/api/admin/vendor-applications?after_id=-1", nil, 422)
	request("admin", "GET", "/api/admin/vendor-applications?after_id=invalid", nil, 422)
}
