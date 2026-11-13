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
	webassets "github.com/wanjiru-noreen/halisi/web"
)

var errResourceForbidden = errors.New("resource or action is not permitted")
var errResourceConflict = errors.New("resource is in use or already exists")
var pricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)

type Resources struct {
	auth    *Auth
	queries *dbsqlc.Queries
}

func NewResources(pool *pgxpool.Pool, auth *Auth) *Resources {
	return &Resources{auth: auth, queries: dbsqlc.New(pool)}
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
	mount("GET /api/customer/orders", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.CustomerOrders(r.Context(), id)
		return map[string]any{"orders": v}, e
	})
	mount("GET /api/customer/orders/{id}", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.CustomerOrder(r.Context(), dbsqlc.CustomerOrderParams{ID: id, CustomerID: owner})
		return map[string]any{"order": v}, e
	})
	mount("GET /api/customer/addresses", models.CustomerResources, func(w http.ResponseWriter, r *http.Request, id int64) (any, error) {
		v, e := h.queries.ListUserAddresses(r.Context(), id)
		return map[string]any{"addresses": v}, e
	})
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
	mount("GET /api/rider/deliveries/{id}", models.RiderAssignments, func(w http.ResponseWriter, r *http.Request, owner int64) (any, error) {
		id, e := resourceID(r)
		if e != nil {
			return nil, e
		}
		v, e := h.queries.AssignedDelivery(r.Context(), dbsqlc.AssignedDeliveryParams{ID: id, RiderID: pgtype.Int8{Int64: owner, Valid: true}})
		return map[string]any{"delivery": v}, e
	})
	mount("PATCH /api/rider/deliveries/{id}", models.RiderAssignments, h.updateDelivery)
	mount("GET /api/admin/users", models.Administration, func(w http.ResponseWriter, r *http.Request, _ int64) (any, error) {
		v, e := h.queries.ManagementUsers(r.Context())
		return map[string]any{"users": v}, e
	})
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
		mux.Handle("GET "+path, h.protect(permission, page))
	}
}

func (h *Resources) protect(permission models.Permission, next http.Handler) http.Handler {
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
	Name  string `json:"name"`
	Price string `json:"price"`
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
	price, err := input.numeric()
	if err != nil {
		return nil, err
	}
	v, err := h.queries.CreateOwnedProduct(r.Context(), dbsqlc.CreateOwnedProductParams{UserID: id, Name: input.Name, Price: price})
	return map[string]any{"product": v}, err
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
	v, err := h.queries.UpdateOwnedProduct(r.Context(), dbsqlc.UpdateOwnedProductParams{ID: id, UserID: owner, Name: input.Name, Price: price})
	return map[string]any{"product": v}, err
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
	v, err := h.queries.UpdateAssignedDelivery(r.Context(), dbsqlc.UpdateAssignedDeliveryParams{ID: id, RiderID: pgtype.Int8{Int64: owner, Valid: true}, Status: input.Status})
	return map[string]any{"delivery": v}, err
}
func (h *Resources) vendorStatus(w http.ResponseWriter, r *http.Request, _ int64) (any, error) {
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
	switch input.Status {
	case "pending", "approved", "rejected", "suspended":
	default:
		return nil, &resourceValidation{"Unsupported vendor status."}
	}
	v, err := h.queries.ManagementVendorStatus(r.Context(), dbsqlc.ManagementVendorStatusParams{ID: id, Status: input.Status})
	return map[string]any{"vendor": v}, err
}
