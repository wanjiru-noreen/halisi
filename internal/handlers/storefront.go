package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type Storefront struct{ pool *pgxpool.Pool }

func NewStorefront(pool *pgxpool.Pool) *Storefront { return &Storefront{pool: pool} }

type publicVendor struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ServiceArea  string `json:"service_area"`
	ProductCount int64  `json:"product_count"`
}

type publicProduct struct {
	ID                int64  `json:"id"`
	VendorID          int64  `json:"vendor_id"`
	VendorName        string `json:"vendor_name"`
	Name              string `json:"name"`
	Price             string `json:"price"`
	AvailableQuantity int32  `json:"available_quantity"`
}

func (s *Storefront) Routes(mux *http.ServeMux) {
	mux.Handle("GET /customer/vendors", webassets.Page("customer/vendors"))
	mux.Handle("GET /customer/products", webassets.Page("customer/products"))
	mux.HandleFunc("GET /api/storefront/vendors", s.listVendors)
	mux.HandleFunc("GET /api/storefront/products", s.listProducts)
}

func (s *Storefront) listProducts(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if !utf8.ValidString(query) || len(query) > 100 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Search must be 100 characters or fewer."})
		return
	}
	var vendorID int64
	if raw := r.URL.Query().Get("vendor_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Vendor ID must be a positive integer."})
			return
		}
		vendorID = id
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, v.id, v.name, p.name, p.price::text,
		       COALESCE(vi.quantity - vi.reserved_quantity, 0)::integer
		FROM products p
		JOIN vendors v ON v.id = p.vendor_id
		LEFT JOIN vendor_inventory vi ON vi.product_id = p.id
		WHERE v.status = 'approved'
		  AND ($1 = 0 OR v.id = $1)
		  AND ($2 = '' OR p.name ILIKE '%' || $2 || '%' OR v.name ILIKE '%' || $2 || '%')
		ORDER BY v.name, p.name, p.id
		LIMIT 100`, vendorID, query)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product listings are temporarily unavailable."})
		return
	}
	defer rows.Close()
	products := make([]publicProduct, 0)
	for rows.Next() {
		var p publicProduct
		if err := rows.Scan(&p.ID, &p.VendorID, &p.VendorName, &p.Name, &p.Price, &p.AvailableQuantity); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product listings are temporarily unavailable."})
			return
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product listings are temporarily unavailable."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

func (s *Storefront) listVendors(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if !utf8.ValidString(query) || len(query) > 100 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Search must be 100 characters or fewer."})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		SELECT v.id, v.name, COALESCE(a.city, v.address) AS service_area,
		       COUNT(vi.product_id)::bigint AS product_count
		FROM vendors v
		LEFT JOIN addresses a ON a.id = v.address_id
		LEFT JOIN products p ON p.vendor_id = v.id
		LEFT JOIN vendor_inventory vi ON vi.product_id = p.id AND vi.quantity > vi.reserved_quantity
		WHERE v.status = 'approved'
		  AND ($1 = '' OR v.name ILIKE '%' || $1 || '%' OR COALESCE(a.city, v.address) ILIKE '%' || $1 || '%')
		GROUP BY v.id, v.name, a.city, v.address
		ORDER BY v.name, v.id
		LIMIT 50`, query)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Vendor listings are temporarily unavailable."})
		return
	}
	defer rows.Close()
	vendors := make([]publicVendor, 0)
	for rows.Next() {
		var v publicVendor
		if err := rows.Scan(&v.ID, &v.Name, &v.ServiceArea, &v.ProductCount); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Vendor listings are temporarily unavailable."})
			return
		}
		vendors = append(vendors, v)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Vendor listings are temporarily unavailable."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": vendors})
}
