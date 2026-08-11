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

		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		role := r.FormValue("role")

		if role != "customer" && role != "owner" {
			http.Error(w, "Invalid account type", http.StatusBadRequest)
			return
		}

		user := models.User{
			Name:     r.FormValue("name"),
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
			Role:     role,
		}

		registeredUser, err := h.userService.RegisterUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if role == "owner" {
			ownerID := registeredUser.ID

			shop := models.Shop{
				Name:     r.FormValue("shop_name"),
				Location: r.FormValue("location"),
				Phone:    r.FormValue("phone"),
				OwnerID:  &ownerID,
			}

			_, err := h.shopService.CreateShop(shop)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Login page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/login.html"))

		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

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

		switch user.Role {
		case "owner":
			http.Redirect(w, r, "/shop/orders", http.StatusSeeOther)
		case "admin":
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
		default:
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/dashboard.html"))

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	tmpl := template.Must(
		template.ParseFiles(
			"web/templates/navbar.html",
			"web/templates/shops.html",
		),
	)

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

// Order page
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

				if f.IsValid() &&
					f.Kind() >= reflect.Int &&
					f.Kind() <= reflect.Int64 {
					return int(f.Int())
				}
			}

			if v.IsValid() && v.Kind() == reflect.Map {
				key := reflect.ValueOf("SelectedShopID")
				val := v.MapIndex(key)

				if val.IsValid() &&
					val.Kind() >= reflect.Int &&
					val.Kind() <= reflect.Int64 {
					return int(val.Int())
				}
			}

			return 0
		}

		funcMap := template.FuncMap{
			"selectedShopID": getSelectedShopID,
		}

		tmpl := template.Must(
			template.New("order.html").
				Funcs(funcMap).
				ParseFiles(
					"web/templates/navbar.html",
					"web/templates/order.html",
				),
		)

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

		shop, err := h.shopService.GetShopByID(shopID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if shop.ID == 0 {
			http.Error(w, "Shop not found", http.StatusNotFound)
			return
		}

		quantity, err := strconv.Atoi(r.FormValue("quantity"))
		if err != nil || quantity <= 0 {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		cylinderSize := r.FormValue("cylinder_size")

		var price float64

		switch cylinderSize {
		case "6kg":
			price = shop.Price6kg
		case "13kg":
			price = shop.Price13kg
		case "45kg":
			price = 6000
		default:
			http.Error(w, "Invalid cylinder size", http.StatusBadRequest)
			return
		}

		if price <= 0 {
			http.Error(w, "Selected gas size is currently unavailable", http.StatusBadRequest)
			return
		}

		order := models.Order{
			UserID:          userID,
			ShopID:          shopID,
			CylinderSize:    cylinderSize,
			Quantity:        quantity,
			TotalPrice:      price * float64(quantity),
			Status:          "Pending",
			DeliveryAddress: r.FormValue("address"),
			ShopName:        shop.Name,
		}

		placedOrder, err := h.orderService.CreateOrder(order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(
			template.ParseFiles(
				"web/templates/navbar.html",
				"web/templates/order_confirmation.html",
			),
		)

		if err := tmpl.ExecuteTemplate(
			w,
			"order_confirmation.html",
			placedOrder,
		); err != nil {
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

	tmpl := template.Must(
		template.ParseFiles(
			"web/templates/navbar.html",
			"web/templates/orders.html",
		),
	)

	if err := tmpl.ExecuteTemplate(w, "orders.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Logout
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

// Update order status
func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ownerID, ok := middleware.GetCurrentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderService.GetOrderByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if order.ID == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	shop, err := h.shopService.GetShopByOwnerID(ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if shop.ID == 0 || order.ShopID != shop.ID {
		http.Error(w, "You are not authorized to update this order", http.StatusForbidden)
		return
	}

	status := r.FormValue("status")

	if status == "" {
		http.Error(w, "Invalid order status", http.StatusBadRequest)
		return
	}

	err = h.orderService.UpdateOrderStatus(id, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/shop/orders", http.StatusSeeOther)
}

// Shop owner orders
func (h *Handler) ShopOrders(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetCurrentUserID(r)

	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	shop, err := h.shopService.GetShopByOwnerID(ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if shop.ID == 0 {
		http.Error(w, "No shop found for this owner", http.StatusNotFound)
		return
	}

	orders, err := h.orderService.GetOrdersByShopID(shop.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Shop   models.Shop
		Orders []models.Order
	}{
		Shop:   shop,
		Orders: orders,
	}

	tmpl := template.Must(
		template.ParseFiles("web/templates/shop_orders.html"),
	)

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Shop management
func (h *Handler) ManageShop(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.GetCurrentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	shop, err := h.shopService.GetShopByOwnerID(ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if shop.ID == 0 {
		http.Error(w, "No shop found for this owner", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(
			template.ParseFiles("web/templates/shop_manage.html"),
		)

		if err := tmpl.Execute(w, shop); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		price6kg, err := strconv.ParseFloat(
			r.FormValue("price_6kg"),
			64,
		)
		if err != nil || price6kg < 0 {
			http.Error(w, "Invalid 6kg price", http.StatusBadRequest)
			return
		}

		price13kg, err := strconv.ParseFloat(
			r.FormValue("price_13kg"),
			64,
		)
		if err != nil || price13kg < 0 {
			http.Error(w, "Invalid 13kg price", http.StatusBadRequest)
			return
		}

		err = h.shopService.UpdateShopPrices(
			shop.ID,
			price6kg,
			price13kg,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/shop/manage", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Delete order
func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetCurrentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderService.GetOrderByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if order.ID == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if order.UserID != userID {
		http.Error(w, "You are not authorized to delete this order", http.StatusForbidden)
		return
	}

	err = h.orderService.DeleteOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// Admin dashboard
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

	tmpl := template.Must(
		template.ParseFiles("web/templates/admin.html"),
	)

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
