package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"halisi/internal/models"
	"halisi/internal/services"
)

type Handler struct {
	userService  *services.UserService
	shopService  *services.ShopService
	orderService *services.OrderService
}

func NewHandler(
	userService *services.UserService,
	shopService *services.ShopService,
	orderService *services.OrderService,
) *Handler {
	return &Handler{
		userService:  userService,
		shopService:  shopService,
		orderService: orderService,
	}
}

// Home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Register page
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/register.html"))
		tmpl.Execute(w, nil)

	case http.MethodPost:

		user := models.User{
			Name:     r.FormValue("name"),
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
			Role:     "customer",
		}

		_, err := h.userService.RegisterUser(user)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Login page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/login.html"))
		tmpl.Execute(w, nil)

	case http.MethodPost:

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := h.userService.LoginUser(email)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if user.ID == 0 {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		if user.Password != password {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/dashboard.html"))
	tmpl.Execute(w, nil)
}

// Shops page
func (h *Handler) Shops(w http.ResponseWriter, r *http.Request) {

	shops, err := h.shopService.GetShops()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/shops.html"))

	err = tmpl.Execute(w, shops)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Shop page
func (h *Handler) Shop(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		http.Error(w, "Invalid shop ID", http.StatusBadRequest)
		return
	}

	shop, err := h.shopService.GetShopByID(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if shop.ID == 0 {
		http.Error(w, "Shop not found", http.StatusNotFound)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/shop.html"))

	err = tmpl.Execute(w, shop)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Order page
func (h *Handler) Order(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/order.html"))
		tmpl.Execute(w, nil)

	case http.MethodPost:

		quantity, err := strconv.Atoi(r.FormValue("quantity"))

		if err != nil {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		order := models.Order{
			UserID:       1,
			ShopID:       1,
			CylinderSize: r.FormValue("cylinder_size"),
			Quantity:     quantity,
			TotalPrice:   1000 * float64(quantity),
			Status:       "Pending",
		}

		_, err = h.orderService.CreateOrder(order)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/orders", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Orders page
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {

	orders, err := h.orderService.GetOrders()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/orders.html"))

	tmpl.Execute(w, orders)
}
