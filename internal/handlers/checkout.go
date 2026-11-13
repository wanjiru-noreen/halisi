package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type Checkout struct {
	pool *pgxpool.Pool
	auth *Auth
}

func NewCheckout(pool *pgxpool.Pool, auth *Auth) *Checkout {
	return &Checkout{pool: pool, auth: auth}
}

func (c *Checkout) Routes(mux *http.ServeMux) {
	mux.Handle("GET /customer/cart", webassets.Page("customer/cart"))
	mux.Handle("POST /api/customer/checkout", middleware.RequireAuth(c.auth.service, c.auth.cookieName, middleware.RequirePermission(models.CustomerResources, http.HandlerFunc(c.submit))))
}

type checkoutItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}

type checkoutInput struct {
	AddressID int64          `json:"address_id"`
	Items     []checkoutItem `json:"items"`
}

type checkoutProduct struct {
	ID, VendorID int64
	Name         string
	Price        string
	Quantity     int32
	Reserved     int32
}

const maxOrderCents int64 = 999999999999

func (c *Checkout) submit(w http.ResponseWriter, r *http.Request) {
	var input checkoutInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.AddressID <= 0 || len(input.Items) == 0 || len(input.Items) > 50 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Choose a delivery address and between 1 and 50 products."})
		return
	}
	quantities := make(map[int64]int32, len(input.Items))
	ids := make([]int64, 0, len(input.Items))
	for _, item := range input.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 || item.Quantity > 100 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Product quantities must be between 1 and 100."})
			return
		}
		if _, exists := quantities[item.ProductID]; exists {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Each product can appear only once in the cart."})
			return
		}
		quantities[item.ProductID] = item.Quantity
		ids = append(ids, item.ProductID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	defer tx.Rollback(ctx)
	var addressLine, city string
	var latitude, longitude pgtype.Numeric
	err = tx.QueryRow(ctx, `SELECT address_line, city, latitude, longitude FROM addresses WHERE id=$1 AND user_id=$2`, input.AddressID, user.ID).Scan(&addressLine, &city, &latitude, &longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Choose one of your saved delivery addresses."})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT p.id, p.vendor_id, p.name, p.price::text, vi.quantity, vi.reserved_quantity
		FROM products p
		JOIN vendors v ON v.id=p.vendor_id AND v.status='approved' AND v.is_active
		JOIN vendor_inventory vi ON vi.product_id=p.id
		WHERE p.id = ANY($1)
		ORDER BY p.id
		FOR UPDATE OF vi`, ids)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	products := make(map[int64]checkoutProduct, len(ids))
	for rows.Next() {
		var p checkoutProduct
		if err := rows.Scan(&p.ID, &p.VendorID, &p.Name, &p.Price, &p.Quantity, &p.Reserved); err != nil {
			rows.Close()
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
			return
		}
		products[p.ID] = p
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	if len(products) != len(ids) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "A product is no longer available from an approved vendor. Refresh your cart."})
		return
	}
	var vendorID int64
	var totalCents int64
	prices := make(map[int64]int64, len(products))
	for _, id := range ids {
		p := products[id]
		if vendorID == 0 {
			vendorID = p.VendorID
		} else if p.VendorID != vendorID {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Checkout is limited to products from one vendor. Empty your cart and choose a single shop."})
			return
		}
		quantity := quantities[id]
		if p.Quantity-p.Reserved < quantity {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "A product has insufficient stock. Refresh your cart and adjust quantities."})
			return
		}
		price, err := decimalCents(p.Price)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "A product price could not be processed."})
			return
		}
		line := price * int64(quantity)
		if line > maxOrderCents-totalCents {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Cart total exceeds the maximum order amount."})
			return
		}
		totalCents += line
		prices[id] = price
	}
	total := centsDecimal(totalCents)
	var pgTotal pgtype.Numeric
	if err := pgTotal.Scan(total); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	deliveryAddress := strings.TrimSpace(addressLine + ", " + city)
	var orderID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer_id,vendor_id,address_id,delivery_address,total,currency,delivery_latitude,delivery_longitude)
		VALUES ($1,$2,$3,$4,$5,'KES',$6,$7) RETURNING id`,
		user.ID, vendorID, pgtype.Int8{Int64: input.AddressID, Valid: true}, deliveryAddress, pgTotal, latitude, longitude).Scan(&orderID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	for _, id := range ids {
		price := centsDecimal(prices[id])
		var unitPrice pgtype.Numeric
		if err := unitPrice.Scan(price); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
			return
		}
		if _, err := tx.Exec(ctx, `INSERT INTO order_items (order_id,vendor_id,product_id,quantity,unit_price) VALUES ($1,$2,$3,$4,$5)`, orderID, vendorID, id, quantities[id], unitPrice); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
			return
		}
		tag, err := tx.Exec(ctx, `UPDATE vendor_inventory SET reserved_quantity=reserved_quantity+$2 WHERE product_id=$1 AND quantity-reserved_quantity >= $2`, id, quantities[id])
		if err != nil || tag.RowsAffected() != 1 {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Stock changed during checkout. Refresh your cart and try again."})
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Checkout is temporarily unavailable."})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": map[string]any{"id": orderID, "status": "pending", "currency": "KES", "total": total}})
}

func decimalCents(raw string) (int64, error) {
	whole, fraction, hasFraction := strings.Cut(raw, ".")
	if !hasFraction {
		fraction = ""
	}
	if whole == "" || len(fraction) > 2 || strings.HasPrefix(whole, "-") {
		return 0, fmt.Errorf("invalid decimal")
	}
	if len(fraction) == 0 {
		fraction = "00"
	} else if len(fraction) == 1 {
		fraction += "0"
	}
	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || units > maxOrderCents/100 {
		return 0, fmt.Errorf("decimal out of range")
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid decimal")
	}
	return units*100 + cents, nil
}

func centsDecimal(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
