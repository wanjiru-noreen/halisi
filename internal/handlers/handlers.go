package handlers

import (
	"html/template"
	"net/http"
	"reflect"
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

	data := struct {
		Shops []models.Shop
	}{
		Shops: shops,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/navbar.html", "web/templates/shops.html"))

	if err := tmpl.ExecuteTemplate(w, "shops.html", data); err != nil {
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

		selectedID := 0
		if shopID := r.URL.Query().Get("shop_id"); shopID != "" {
			if id, err := strconv.Atoi(shopID); err == nil {
				selectedID = id
			}
		}

		data := struct {
			Shops          []models.Shop
			SelectedShopID int
		}{
			Shops:          shops,
			SelectedShopID: selectedID,
		}

		// helper to safely read SelectedShopID from the root data without panicking
		getSelectedShopID := func(root interface{}) int {
			v := reflect.ValueOf(root)
			if !v.IsValid() {
				return 0
			}
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.IsValid() && v.Kind() == reflect.Struct {
				f := v.FieldByName("SelectedShopID")
				if f.IsValid() && f.Kind() >= reflect.Int && f.Kind() <= reflect.Int64 {
					return int(f.Int())
				}
			}
			if v.IsValid() && v.Kind() == reflect.Map {
				key := reflect.ValueOf("SelectedShopID")
				val := v.MapIndex(key)
				if val.IsValid() && val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64 {
					return int(val.Int())
				}
			}
			return 0
		}

		funcMap := template.FuncMap{"selectedShopID": getSelectedShopID}

		tmpl := template.Must(template.New("order.html").Funcs(funcMap).ParseFiles("web/templates/navbar.html", "web/templates/order.html"))

		if err := tmpl.ExecuteTemplate(w, "order.html", data); err != nil {
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

		cylinderSize := r.FormValue("cylinder_size")
		price := 0.0

		switch cylinderSize {
		case "6kg":
			price = 1400
		case "13kg":
			price = 2800
		case "45kg":
			price = 6000
		default:
			http.Error(w, "Invalid cylinder size", http.StatusBadRequest)
			return
		}

		order := models.Order{
			UserID:          userID,
			ShopID:          shopID,
			CylinderSize:    cylinderSize,
			Quantity:        quantity,
			TotalPrice:      price * float64(quantity),
			Status:          "On its way",
			DeliveryAddress: r.FormValue("address"),
		}

		placedOrder, err := h.orderService.CreateOrder(order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		shop, err := h.shopService.GetShopByID(shopID)
		if err == nil && shop.ID != 0 {
			placedOrder.ShopName = shop.Name
		}

		tmpl := template.Must(template.ParseFiles("web/templates/navbar.html", "web/templates/order_confirmation.html"))
		if err := tmpl.ExecuteTemplate(w, "order_confirmation.html", placedOrder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

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

	data := struct {
		Orders []models.Order
	}{
		Orders: orders,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/navbar.html", "web/templates/orders.html"))

	if err := tmpl.ExecuteTemplate(w, "orders.html", data); err != nil {
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
