package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

var errResourceForbidden = errors.New("resource or action is not permitted")
var errResourceConflict = errors.New("resource is in use or already exists")
var pricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)

type Resources struct {
	auth       *Auth
	pool       *pgxpool.Pool
	queries    *dbsqlc.Queries
	onboarding *services.VendorOnboarding
	riders     *services.RiderRegistration
}

func NewResources(pool *pgxpool.Pool, auth *Auth, onboarding ...*services.VendorOnboarding) *Resources {
	h := &Resources{auth: auth, pool: pool, queries: dbsqlc.New(pool), riders: services.NewRiderRegistration(pool)}
	if len(onboarding) > 0 {
		h.onboarding = onboarding[0]
	}
	return h
}

// Each endpoint receives the session principal, never a caller-supplied owner ID.
type scopedEndpoint func(http.ResponseWriter, *http.Request, int64) (any, error)

func (h *Resources) Routes(mux *http.ServeMux) {
	mount := func(pattern string, permission models.Permission, endpoint scopedEndpoint) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			user, _ := middleware.CurrentUser(ctx)
			data, err := endpoint(w, r.WithContext(ctx), user.ID)
			if err != nil {
				h.resourceError(w, err)
				return
			}
			if data != nil {
				writeJSON(w, 200, data)
			}
		})
		mux.Handle(pattern, h.protect(permission, handler))
	}
	mount("GET /api/customer/orders", models.CustomerResources, h.listCustomerOrders)
	mount("GET /api/customer/deliveries/{id}", models.CustomerResources, h.getDelivery)
	mount("GET /api/vendor/deliveries/{id}", models.VendorResources, h.getDelivery)
	mount("GET /api/admin/deliveries/{id}", models.Administration, h.getDelivery)
	mount("POST /api/vendor/orders/{id}/confirm", models.VendorResources, h.confirmOrder)
	mount("GET /api/customer/orders/{id}", models.CustomerResources, h.getCustomerOrder)
	mount("GET /api/customer/addresses", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.ListUserAddresses(r.Context(), id)
		return map[string]any{"addresses": v}, e
	})
	mount("POST /api/customer/addresses", models.CustomerResources, h.createAddress)
	mount("PATCH /api/customer/addresses/{id}", models.CustomerResources, h.updateAddress)
	mount("DELETE /api/customer/addresses/{id}", models.CustomerResources, h.deleteAddress)
	mount("GET /api/customer/addresses/{id}", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.CustomerAddress(r.Context(), dbsqlc.CustomerAddressParams{ID: id, UserID: owner})
		return map[string]any{"address": v}, e
	})
	mount("GET /api/customer/payments", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.CustomerPayments(r.Context(), id)
		return map[string]any{"payments": v}, e
	})
	mount("GET /api/customer/payments/{id}", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.CustomerPayment(r.Context(), dbsqlc.CustomerPaymentParams{ID: id, CustomerID: owner})
		return map[string]any{"payment": v}, e
	})
	mount("GET /api/vendor/shop", models.VendorResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.OwnedShop(r.Context(), id)
		return map[string]any{"shop": v}, e
	})
	mount("POST /api/vendor/shop", models.VendorResources, h.createShop)
	mount("PATCH /api/vendor/shop", models.VendorResources, h.updateShop)
	mount("GET /api/vendor/products", models.VendorResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.OwnedProducts(r.Context(), id)
		return map[string]any{"products": v}, e
	})
	mount("GET /api/vendor/products/{id}", models.VendorResources, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.OwnedProduct(r.Context(), dbsqlc.OwnedProductParams{ID: id, UserID: owner})
		return map[string]any{"product": v}, e
	})
	mount("POST /api/vendor/products", models.VendorResources, h.createProduct)
	mount("PATCH /api/vendor/products/{id}", models.VendorResources, h.updateProduct)
	mount("DELETE /api/vendor/products/{id}", models.VendorResources, h.deleteProduct)
	mount("GET /api/vendor/orders", models.VendorResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.VendorOrders(r.Context(), id)
		return map[string]any{"orders": v}, e
	})
	mount("GET /api/vendor/orders/{id}", models.VendorResources, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.VendorOrder(r.Context(), dbsqlc.VendorOrderParams{ID: id, UserID: owner})
		return map[string]any{"order": v}, e
	})
	mount("GET /api/rider/deliveries", models.RiderAssignments, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.AssignedDeliveries(r.Context(), pgtype.Int8{Int64: id, Valid: true})
		return map[string]any{"deliveries": v}, e
	})
	mount("GET /api/rider/deliveries/{id}", models.RiderAssignments, h.getDelivery)
	mount("PATCH /api/rider/deliveries/{id}", models.RiderAssignments, h.updateDelivery)
	mount("GET /api/admin/users", models.Administration, func(w http.ResponseWriter, r *http.Request, _ int64) (any, error) {
		v, e := h.queries.ManagementUsers(r.Context())
		return map[string]any{"users": v}, e
	})
	mount("GET /api/admin/dashboard", models.Administration, h.adminDashboard)
	mount("GET /api/admin/vendors", models.Administration, func(w http.ResponseWriter, r *http.Request, _ int64) (any, error) {
		v, e := h.queries.ManagementVendors(r.Context())
		return map[string]any{"vendors": v}, e
	})
	mount("PATCH /api/admin/vendors/{id}/status", models.Administration, h.vendorStatus)

	mux.Handle("GET /dashboard", middleware.RequireAuth(h.auth.service, h.auth.cookieName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := middleware.CurrentUser(r.Context())
		path, ok := models.DashboardPath(models.Role(user.Role))
		if !ok {
			h.resourceError(w, errResourceForbidden)
			return
		}
		http.Redirect(w, r, path, http.StatusSeeOther)
	})))
	for role, permission := range map[models.Role]models.Permission{models.Customer: models.CustomerResources, models.Vendor: models.VendorResources, models.Rider: models.RiderAssignments, models.Administrator: models.Administration} {
		path, _ := models.DashboardPath(role)
		page := webassets.Page("dashboard")
		if role == models.Customer {
			page = webassets.Page("customer/dashboard")
		}
		if role == models.Administrator {
			page = webassets.Page("admin/dashboard")
		}
		mux.Handle("GET "+path, h.protect(permission, page))
	}
	mux.Handle("GET /customer/addresses", h.protect(models.CustomerResources, webassets.Page("customer/addresses")))
	mux.Handle("GET /customer/orders", h.protect(models.CustomerResources, webassets.Page("customer/orders")))
	mux.Handle("GET /customer/orders/{id}", h.protect(models.CustomerResources, webassets.Page("customer/order-detail")))
}

type customerOrderView struct {
	ID              int64     `json:"id"`
	VendorID        int64     `json:"vendor_id"`
	VendorName      string    `json:"vendor_name"`
	Status          string    `json:"status"`
	Total           string    `json:"total"`
	Currency        string    `json:"currency"`
	DeliveryAddress string    `json:"delivery_address"`
	CreatedAt       time.Time `json:"created_at"`
}

type customerOrderItemView struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int32  `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
}

type deliveryView struct {
	ID          int64      `json:"id"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at"`
}

func (h *Resources) listCustomerOrders(_ http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT o.id,o.vendor_id,v.name,o.status,o.total::text,o.currency,o.delivery_address,o.created_at
		FROM orders o JOIN vendors v ON v.id=o.vendor_id
		WHERE o.customer_id=$1 ORDER BY o.id DESC LIMIT 100`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]customerOrderView, 0)
	for rows.Next() {
		var order customerOrderView
		if err := rows.Scan(&order.ID, &order.VendorID, &order.VendorName, &order.Status, &order.Total, &order.Currency, &order.DeliveryAddress, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"orders": orders}, nil
}

func (h *Resources) getCustomerOrder(_ http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var order customerOrderView
	err = h.pool.QueryRow(r.Context(), `
		SELECT o.id,o.vendor_id,v.name,o.status,o.total::text,o.currency,o.delivery_address,o.created_at
		FROM orders o JOIN vendors v ON v.id=o.vendor_id WHERE o.id=$1 AND o.customer_id=$2`, id, owner).
		Scan(&order.ID, &order.VendorID, &order.VendorName, &order.Status, &order.Total, &order.Currency, &order.DeliveryAddress, &order.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errResourceForbidden
	}
	if err != nil {
		return nil, err
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT oi.product_id,p.name,oi.quantity,oi.unit_price::text
		FROM order_items oi JOIN products p ON p.id=oi.product_id
		WHERE oi.order_id=$1 ORDER BY oi.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]customerOrderItemView, 0)
	for rows.Next() {
		var item customerOrderItemView
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var delivery *deliveryView
	var deliveredAt pgtype.Timestamptz
	var d deliveryView
	err = h.pool.QueryRow(r.Context(), `SELECT id,status,delivered_at FROM deliveries WHERE order_id=$1`, id).Scan(&d.ID, &d.Status, &deliveredAt)
	if err == nil {
		if deliveredAt.Valid {
			value := deliveredAt.Time
			d.DeliveredAt = &value
		}
		delivery = &d
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return map[string]any{"order": order, "items": items, "delivery": delivery}, nil
}

func (h *Resources) protect(permission models.Permission, next http.Handler) http.Handler {
	if permission == models.RiderAssignments {
		next = middleware.RequireApprovedRider(h.riders, next)
	}
	return middleware.RequireAuth(h.auth.service, h.auth.cookieName, middleware.RequirePermission(permission, next))
}

func resourceID(r *http.Request) (int64, error) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id <= 0 {
		return 0, &resourceValidation{"Resource ID must be a positive integer."}
	}
	return id, nil
}

type resourceValidation struct{ message string }

func (e *resourceValidation) Error() string { return e.message }

func (h *Resources) resourceError(w http.ResponseWriter, err error) {
	var validation *resourceValidation
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, errResourceForbidden):
		writeJSON(w, 403, map[string]string{"error": "Resource or action is not permitted."})
	case errors.As(err, &validation):
		writeJSON(w, 422, map[string]string{"error": validation.Error()})
	case errors.Is(err, errResourceConflict):
		writeJSON(w, 409, map[string]string{"error": "Resource is in use or already exists."})
	case errors.Is(err, services.ErrVendorReviewStale):
		writeJSON(w, 409, map[string]string{"error": err.Error()})
	case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23505"):
		writeJSON(w, 409, map[string]string{"error": "Resource is in use or already exists."})
	case errors.As(err, &pgErr) && (pgErr.Code == "23514" || pgErr.Code == "22003"):
		writeJSON(w, 422, map[string]string{"error": "Invalid resource values."})
	default:
		writeJSON(w, 503, map[string]string{"error": "The resource is temporarily unavailable."})
	}
}

type shopInput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (input *shopInput) validate() error {
	input.Name = strings.TrimSpace(input.Name)
	input.Address = strings.TrimSpace(input.Address)
	if !validName(input.Name) || len(input.Address) == 0 || len(input.Address) > 500 || strings.ContainsFunc(input.Address, unicode.IsControl) {
		return &resourceValidation{"Provide a name (2–100 characters) and address (1–500 bytes)."}
	}
	return nil
}
func validName(name string) bool {
	return utf8.ValidString(name) && utf8.RuneCountInString(name) >= 2 && utf8.RuneCountInString(name) <= 100 && !strings.ContainsFunc(name, unicode.IsControl)
}
func (h *Resources) createShop(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
	var input shopInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	v, err := h.queries.CreateOwnedShop(r.Context(), dbsqlc.CreateOwnedShopParams{UserID: id, Name: input.Name, Address: input.Address})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errResourceConflict
	}
	return map[string]any{"shop": v}, err
}
func (h *Resources) updateShop(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
	var input shopInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	v, err := h.queries.UpdateOwnedShop(r.Context(), dbsqlc.UpdateOwnedShopParams{UserID: id, Name: input.Name, Address: input.Address})
	return map[string]any{"shop": v}, err
}

type productInput struct {
	Name     string `json:"name"`
	Price    string `json:"price"`
	Quantity *int32 `json:"quantity"`
}

func (input *productInput) numeric() (pgtype.Numeric, error) {
	input.Name = strings.TrimSpace(input.Name)
	var p pgtype.Numeric
	if !validName(input.Name) || !pricePattern.MatchString(input.Price) {
		return p, &resourceValidation{"Provide a name (2–100 characters) and a nonnegative decimal price string with up to two decimal places."}
	}
	if err := p.Scan(input.Price); err != nil {
		return p, err
	}
	return p, nil
}
func (h *Resources) createProduct(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
	var input productInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if input.Quantity == nil || *input.Quantity < 1 || *input.Quantity > 100000 {
		return nil, &resourceValidation{"Initial stock quantity must be between 1 and 100000."}
	}
	price, err := input.numeric()
	if err != nil {
		return nil, err
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(r.Context())
	q := h.queries.WithTx(tx)
	product, err := q.CreateOwnedProduct(r.Context(), dbsqlc.CreateOwnedProductParams{UserID: id, Name: input.Name, Price: price})
	if err != nil {
		return nil, err
	}
	if _, err := q.SetInventory(r.Context(), dbsqlc.SetInventoryParams{ProductID: product.ID, VendorID: product.VendorID, Quantity: *input.Quantity}); err != nil {
		return nil, err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return nil, err
	}
	return map[string]any{"product": product}, nil
}
func (h *Resources) updateProduct(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input productInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	price, err := input.numeric()
	if err != nil {
		return nil, err
	}
	if input.Quantity != nil && (*input.Quantity < 0 || *input.Quantity > 100000) {
		return nil, &resourceValidation{"Stock quantity must be between 0 and 100000."}
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(r.Context())
	q := h.queries.WithTx(tx)
	product, err := q.UpdateOwnedProduct(r.Context(), dbsqlc.UpdateOwnedProductParams{ID: id, UserID: owner, Name: input.Name, Price: price})
	if err != nil {
		return nil, err
	}
	if input.Quantity != nil {
		if _, err := q.SetInventory(r.Context(), dbsqlc.SetInventoryParams{ProductID: product.ID, VendorID: product.VendorID, Quantity: *input.Quantity}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return nil, err
	}
	return map[string]any{"product": product}, nil
}
func (h *Resources) deleteProduct(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	count, err := h.queries.DeleteOwnedProduct(r.Context(), dbsqlc.DeleteOwnedProductParams{ID: id, UserID: owner})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errResourceForbidden
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(204)
	return nil, nil
}
func (h *Resources) updateDelivery(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if input.Status != "picked_up" && input.Status != "delivered" && input.Status != "failed" {
		return nil, &resourceValidation{"Unsupported delivery transition."}
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(r.Context())
	q := h.queries.WithTx(tx)
	var currentStatus string
	if err := tx.QueryRow(r.Context(), `SELECT status FROM deliveries WHERE id=$1 AND rider_id=$2 FOR UPDATE`, id, owner).Scan(&currentStatus); err != nil {
		return nil, err
	}
	if !services.DeliveryTransitionAllowed(currentStatus, input.Status) {
		return nil, errResourceForbidden
	}
	delivery, err := q.UpdateAssignedDelivery(r.Context(), dbsqlc.UpdateAssignedDeliveryParams{ID: id, RiderID: pgtype.Int8{Int64: owner, Valid: true}, Status: input.Status})
	if err != nil {
		return nil, err
	}
	if input.Status == "failed" {
		var itemCount int64
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM order_items WHERE order_id=$1`, delivery.OrderID).Scan(&itemCount); err != nil {
			return nil, err
		}
		tag, err := tx.Exec(r.Context(), `
			UPDATE vendor_inventory vi
			SET reserved_quantity=vi.reserved_quantity-oi.quantity
			FROM order_items oi
			WHERE oi.order_id=$1 AND vi.product_id=oi.product_id
			  AND vi.reserved_quantity >= oi.quantity`, delivery.OrderID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != itemCount {
			return nil, errors.New("could not reconcile reserved inventory for failed delivery")
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return nil, err
	}
	return map[string]any{"delivery": delivery}, nil
}
func (h *Resources) vendorStatus(w http.ResponseWriter, r *http.Request, adminID int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input struct {
		Status        string `json:"status"`
		Reason        string `json:"reason"`
		ReviewVersion int64  `json:"review_version"`
	}
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	switch input.Status {
	case "pending", "approved", "rejected", "suspended":
	default:
		return nil, &resourceValidation{"Unsupported vendor status."}
	}
	if len(strings.TrimSpace(input.Reason)) < 3 || h.onboarding == nil {
		return nil, &resourceValidation{"Use the verification workflow with a review reason and complete application."}
	}
	v, err := h.onboarding.Review(r.Context(), adminID, id, input.Status, input.Reason, input.ReviewVersion)
	if err != nil {
		if errors.Is(err, services.ErrVendorReviewStale) {
			return nil, err
		}
		if errors.Is(err, services.ErrVendorForbidden) {
			return nil, errResourceForbidden
		}
		return nil, &resourceValidation{"Application is incomplete or cannot be reviewed in its current state."}
	}
	return map[string]any{"vendor": v}, err
}

type addressInput struct {
	Label       string   `json:"label"`
	AddressLine string   `json:"address_line"`
	City        string   `json:"city"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

func (input *addressInput) validate() error {
	input.Label = strings.TrimSpace(input.Label)
	input.AddressLine = strings.TrimSpace(input.AddressLine)
	input.City = strings.TrimSpace(input.City)
	if input.Label == "" {
		input.Label = "Home"
	}
	if !utf8.ValidString(input.Label) || utf8.RuneCountInString(input.Label) > 100 || strings.ContainsFunc(input.Label, unicode.IsControl) ||
		!utf8.ValidString(input.AddressLine) || len(input.AddressLine) == 0 || len(input.AddressLine) > 500 || strings.ContainsFunc(input.AddressLine, unicode.IsControl) ||
		!utf8.ValidString(input.City) || len(input.City) == 0 || len(input.City) > 100 || strings.ContainsFunc(input.City, unicode.IsControl) {
		return &resourceValidation{"Provide a label (up to 100 characters), street address (up to 500 bytes), and city (up to 100 bytes)."}
	}
	if (input.Latitude == nil) != (input.Longitude == nil) {
		return &resourceValidation{"Provide both coordinates or leave both empty."}
	}
	if input.Latitude != nil && (*input.Latitude < -90 || *input.Latitude > 90 || *input.Longitude < -180 || *input.Longitude > 180) {
		return &resourceValidation{"Coordinates are outside the valid latitude or longitude range."}
	}
	return nil
}

func (input addressInput) coordinates() (pgtype.Numeric, pgtype.Numeric, error) {
	var latitude, longitude pgtype.Numeric
	if input.Latitude == nil {
		return latitude, longitude, nil
	}
	if err := latitude.Scan(strconv.FormatFloat(*input.Latitude, 'f', 6, 64)); err != nil {
		return latitude, longitude, err
	}
	if err := longitude.Scan(strconv.FormatFloat(*input.Longitude, 'f', 6, 64)); err != nil {
		return latitude, longitude, err
	}
	return latitude, longitude, nil
}

func (h *Resources) createAddress(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	var input addressInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	latitude, longitude, err := input.coordinates()
	if err != nil {
		return nil, &resourceValidation{"Coordinates must be valid numbers."}
	}
	address, err := h.queries.CreateAddress(r.Context(), dbsqlc.CreateAddressParams{UserID: owner, Label: input.Label, AddressLine: input.AddressLine, City: input.City, Latitude: latitude, Longitude: longitude})
	return map[string]any{"address": address}, err
}

func (h *Resources) updateAddress(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input addressInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	latitude, longitude, err := input.coordinates()
	if err != nil {
		return nil, &resourceValidation{"Coordinates must be valid numbers."}
	}
	var address dbsqlc.Address
	err = h.pool.QueryRow(r.Context(), `UPDATE addresses SET label=$3,address_line=$4,city=$5,latitude=COALESCE($6,latitude),longitude=COALESCE($7,longitude) WHERE id=$1 AND user_id=$2 RETURNING id,user_id,label,address_line,city,latitude,longitude,created_at,updated_at`, id, owner, input.Label, input.AddressLine, input.City, latitude, longitude).Scan(&address.ID, &address.UserID, &address.Label, &address.AddressLine, &address.City, &address.Latitude, &address.Longitude, &address.CreatedAt, &address.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errResourceForbidden
	}
	return map[string]any{"address": address}, err
}

func (h *Resources) deleteAddress(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM addresses WHERE id=$1 AND user_id=$2`, id, owner)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errResourceForbidden
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}
