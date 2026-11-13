package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
)

type schemaFixture struct {
	customer, owner, vendor, product, order int64
}

func insertID(t *testing.T, ctx context.Context, db *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		t.Fatalf("fixture insert failed: %v", err)
	}
	return id
}

func seedVersionOne(t *testing.T, ctx context.Context, db *pgxpool.Pool) schemaFixture {
	t.Helper()
	f := schemaFixture{}
	f.customer = insertID(t, ctx, db, "INSERT INTO users (name,email,password_hash,role) VALUES ('Customer','customer@example.invalid','test-hash','customer') RETURNING id")
	f.owner = insertID(t, ctx, db, "INSERT INTO users (name,email,password_hash,role) VALUES ('Vendor','vendor@example.invalid','test-hash','vendor') RETURNING id")
	f.vendor = insertID(t, ctx, db, "INSERT INTO vendors (user_id,name,address) VALUES ($1,'Vendor','Nairobi') RETURNING id", f.owner)
	f.product = insertID(t, ctx, db, "INSERT INTO products (vendor_id,name,price,stock) VALUES ($1,'LPG',1000,7) RETURNING id", f.vendor)
	f.order = insertID(t, ctx, db, "INSERT INTO orders (customer_id,vendor_id,total,delivery_address) VALUES ($1,$2,1000,'Nairobi') RETURNING id", f.customer, f.vendor)
	insertID(t, ctx, db, "INSERT INTO order_items (order_id,product_id,quantity,unit_price) VALUES ($1,$2,1,1000) RETURNING id", f.order, f.product)
	return f
}

func testCoreSchema(t *testing.T, ctx context.Context, db *pgxpool.Pool, f schemaFixture) {
	t.Helper()
	var quantity int
	if err := db.QueryRow(ctx, "SELECT quantity FROM vendor_inventory WHERE product_id=$1", f.product).Scan(&quantity); err != nil || quantity != 7 {
		t.Fatal("stock was not migrated to inventory")
	}
	var itemVendor int64
	if err := db.QueryRow(ctx, "SELECT vendor_id FROM order_items WHERE order_id=$1", f.order).Scan(&itemVendor); err != nil || itemVendor != f.vendor {
		t.Fatal("order-item ownership was not backfilled")
	}
	for _, table := range []string{"users", "sessions", "vendors", "vendor_documents", "products", "vendor_inventory", "addresses", "orders", "order_items", "payments", "deliveries", "reviews"} {
		var count int
		if err := db.QueryRow(ctx, "SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 AND column_name IN ('created_at','updated_at')", table).Scan(&count); err != nil || count != 2 {
			t.Fatalf("missing timestamps on %s", table)
		}
	}
	otherOwner := insertID(t, ctx, db, "INSERT INTO users (name,email,password_hash,role) VALUES ('Other','other@example.invalid','test-hash','vendor') RETURNING id")
	otherVendor := insertID(t, ctx, db, "INSERT INTO vendors (user_id,name,address) VALUES ($1,'Other','Mombasa') RETURNING id", otherOwner)
	otherProduct := insertID(t, ctx, db, "INSERT INTO products (vendor_id,name,price) VALUES ($1,'Other LPG',1200) RETURNING id", otherVendor)
	address := insertID(t, ctx, db, "INSERT INTO addresses (user_id,address_line,city) VALUES ($1,'Street','Nairobi') RETURNING id", f.customer)
	rider := insertID(t, ctx, db, "INSERT INTO users (name,email,password_hash,role) VALUES ('Rider','rider@example.invalid','test-hash','rider') RETURNING id")
	tests := []struct {
		name, code, sql string
		args            []any
	}{
		{"unknown role", "23514", "UPDATE users SET role='unknown' WHERE id=$1", []any{f.customer}},
		{"blank name", "23514", "UPDATE users SET name=' ' WHERE id=$1", []any{f.customer}},
		{"duplicate email ignoring case", "23505", "UPDATE users SET email='CUSTOMER@example.invalid' WHERE id=$1", []any{f.owner}},
		{"unknown user", "23503", "INSERT INTO addresses (user_id,address_line,city) VALUES (-1,'Street','Nairobi')", nil},
		{"invalid coordinates", "23514", "UPDATE addresses SET latitude=91,longitude=0 WHERE id=$1", []any{address}},
		{"unpaired coordinates", "23514", "UPDATE addresses SET latitude=1 WHERE id=$1", []any{address}},
		{"expired session", "23514", "INSERT INTO sessions (user_id,token_hash,expires_at) VALUES ($1,repeat('a',64),NOW()-INTERVAL '1 day')", []any{f.customer}},
		{"raw session token", "23514", "INSERT INTO sessions (user_id,token_hash,expires_at) VALUES ($1,'raw-token',NOW()+INTERVAL '1 day')", []any{f.customer}},
		{"negative inventory", "23514", "UPDATE vendor_inventory SET quantity=-1 WHERE product_id=$1", []any{f.product}},
		{"over reservation", "23514", "UPDATE vendor_inventory SET reserved_quantity=8 WHERE product_id=$1", []any{f.product}},
		{"inventory vendor mismatch", "23503", "INSERT INTO vendor_inventory (vendor_id,product_id) VALUES ($1,$2)", []any{f.vendor, otherProduct}},
		{"cross vendor item", "23503", "INSERT INTO order_items (order_id,vendor_id,product_id,quantity,unit_price) VALUES ($1,$2,$3,1,1200)", []any{f.order, f.vendor, otherProduct}},
		{"zero quantity", "23514", "UPDATE order_items SET quantity=0 WHERE order_id=$1", []any{f.order}},
		{"negative price", "23514", "UPDATE products SET price=-1 WHERE id=$1", []any{f.product}},
		{"NaN price", "23514", "UPDATE products SET price='NaN' WHERE id=$1", []any{f.product}},
		{"address owner mismatch", "23503", "INSERT INTO orders (customer_id,vendor_id,address_id,total,delivery_address) VALUES ($1,$2,$3,100,'Street')", []any{f.owner, f.vendor, address}},
		{"unknown order state", "23514", "UPDATE orders SET status='unknown' WHERE id=$1", []any{f.order}},
		{"zero payment", "23514", "INSERT INTO payments (order_id,amount) VALUES ($1,0)", []any{f.order}},
		{"payment currency mismatch", "23503", "INSERT INTO payments (order_id,amount,currency) VALUES ($1,100,'USD')", []any{f.order}},
		{"incomplete delivery", "23514", "INSERT INTO deliveries (order_id,status) VALUES ($1,'delivered')", []any{f.order}},
		{"invalid rating", "23514", "INSERT INTO reviews (order_id,customer_id,vendor_id,rating) VALUES ($1,$2,$3,6)", []any{f.order, f.customer, f.vendor}},
		{"review customer mismatch", "23503", "INSERT INTO reviews (order_id,customer_id,vendor_id,rating) VALUES ($1,$2,$3,5)", []any{f.order, f.owner, f.vendor}},
		{"review vendor mismatch", "23503", "INSERT INTO reviews (order_id,customer_id,vendor_id,rating) VALUES ($1,$2,$3,5)", []any{f.order, f.customer, otherVendor}},
		{"unknown document state", "23514", "INSERT INTO vendor_documents (vendor_id,document_type,storage_key,status) VALUES ($1,'licence','test/key','unknown')", []any{f.vendor}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(ctx, tt.sql, tt.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.code {
				t.Fatalf("expected SQLSTATE %s, got %v", tt.code, err)
			}
		})
	}
	q := dbsqlc.New(db)
	user, err := q.GetUserByEmail(ctx, "CUSTOMER@example.invalid")
	if err != nil || user.ID != f.customer {
		t.Fatal("generated case-insensitive user lookup failed")
	}
	updated, err := q.UpdateUser(ctx, dbsqlc.UpdateUserParams{ID: user.ID, Name: "Updated", Email: user.Email, Phone: pgtype.Text{}})
	if err != nil || !updated.UpdatedAt.Time.After(user.UpdatedAt.Time) {
		t.Fatal("updated_at trigger did not advance on generated update")
	}
	reserved, err := q.ReserveInventory(ctx, dbsqlc.ReserveInventoryParams{ProductID: f.product, VendorID: f.vendor, Quantity: 2})
	if err != nil || reserved.ReservedQuantity != 2 {
		t.Fatal("generated reservation failed")
	}
	if _, err := q.SaveVendorDocument(ctx, dbsqlc.SaveVendorDocumentParams{VendorID: f.vendor, DocumentType: "licence", StorageKey: "test/key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreatePayment(ctx, dbsqlc.CreatePaymentParams{OrderID: f.order, Amount: pgtype.Numeric{Int: nil, Valid: false}, Currency: "KES"}); err == nil {
		t.Fatal("NULL payment amount should be rejected")
	}
	insertID(t, ctx, db, "INSERT INTO payments (order_id,amount) VALUES ($1,1000) RETURNING id", f.order)
	insertID(t, ctx, db, "INSERT INTO deliveries (order_id,rider_id,status,delivered_at) VALUES ($1,$2,'delivered',NOW()) RETURNING id", f.order, rider)
	if _, err := q.CreateReview(ctx, dbsqlc.CreateReviewParams{OrderID: f.order, CustomerID: f.customer, VendorID: f.vendor, Rating: 5, Comment: "Good"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateReview(ctx, dbsqlc.CreateReviewParams{OrderID: f.order, CustomerID: f.customer, VendorID: f.vendor, Rating: 4}); err == nil {
		t.Fatal("duplicate review should be rejected")
	}
	sessionID := insertID(t, ctx, db, "INSERT INTO sessions (user_id,token_hash,expires_at) VALUES ($1,$2,NOW()+INTERVAL '1 day') RETURNING id", f.customer, strings.Repeat("a", 64))
	if session, err := q.GetActiveSession(ctx, strings.Repeat("a", 64)); err != nil || session.ID != sessionID {
		t.Fatal("generated session lookup failed")
	}
	if _, err := q.RevokeSession(ctx, dbsqlc.RevokeSessionParams{ID: sessionID, UserID: f.customer}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.GetActiveSession(ctx, strings.Repeat("a", 64)); err == nil {
		t.Fatal("revoked session must not be active")
	}
}
