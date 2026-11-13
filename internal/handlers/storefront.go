package handlers

import (
	"context"
	"net/http"
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

func (s *Storefront) Routes(mux *http.ServeMux) {
	mux.Handle("GET /customer/vendors", webassets.Page("customer/vendors"))
	mux.HandleFunc("GET /api/storefront/vendors", s.listVendors)
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
