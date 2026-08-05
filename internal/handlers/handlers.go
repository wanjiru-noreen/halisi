package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"halisi/internal/middleware"
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

	if err := tmpl.Execute(w, nil); err != nil {
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

		user, err := h.userService.LoginUser(email, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if user.ID == 0 {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		session, err := middleware.Store.Get(r, "halisi-session")
		if err != nil {
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		session.Values["user_id"] = user.ID
		session.Values["role"] = user.Role

		err = session.Save(r, w)
		if err != nil {
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
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

	if err := tmpl.Execute(w, shops); err != nil {
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

	if err := tmpl.Execute(w, shop); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Order(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:

		shops, err := h.shopService.GetShops()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(template.ParseFiles("web/templates/order.html"))

		if err := tmpl.Execute(w, shops); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:

		userID, ok := middleware.GetCurrentUserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		shopID, err := strconv.Atoi(r.FormValue("shop_id"))
		if err != nil {
			http.Error(w, "Invalid shop", http.StatusBadRequest)
			return
		}

		quantity, err := strconv.Atoi(r.FormValue("quantity"))
		if err != nil {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		order := models.Order{
			UserID:       userID,
			ShopID:       shopID,
			CylinderSize: r.FormValue("cylinder_size"),
			Quantity:     quantity,
			TotalPrice:   float64(quantity) * 1000,
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

	userID, ok := middleware.GetCurrentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	orders, err := h.orderService.GetOrdersByUserID(userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/orders.html"))

	if err := tmpl.Execute(w, orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

	session, err := middleware.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")

	err = h.orderService.UpdateOrderStatus(id, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (h *Handler) ShopOrders(w http.ResponseWriter, r *http.Request) {

	shopID, err := strconv.Atoi(r.URL.Query().Get("shop_id"))
	if err != nil {
		http.Error(w, "Invalid shop ID", http.StatusBadRequest)
		return
	}

	orders, err := h.orderService.GetOrdersByShopID(shopID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/shop_orders.html"))

	err = tmpl.Execute(w, orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	err = h.orderService.DeleteOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shops, err := h.shopService.GetShops()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orders, err := h.orderService.GetOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Users       []models.User
		Shops       []models.Shop
		Orders      []models.Order
		TotalUsers  int
		TotalShops  int
		TotalOrders int
	}{
		Users:       users,
		Shops:       shops,
		Orders:      orders,
		TotalUsers:  len(users),
		TotalShops:  len(shops),
		TotalOrders: len(orders),
	}

	tmpl := template.Must(template.ParseFiles("web/templates/admin.html"))

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
