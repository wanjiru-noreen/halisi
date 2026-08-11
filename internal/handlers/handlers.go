package handlers

import (
	"bytes"
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

func renderTemplate(w http.ResponseWriter, templateName string, data interface{}, files ...string) {
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Failed to load page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer

	if err := tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		http.Error(w, "Failed to render page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	renderTemplate(
		w,
		"index.html",
		nil,
		"web/templates/index.html",
	)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		renderTemplate(
			w,
			"register.html",
			nil,
			"web/templates/register.html",
		)

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

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		renderTemplate(
			w,
			"login.html",
			nil,
			"web/templates/login.html",
		)

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

		if err := session.Save(r, w); err != nil {
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

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	renderTemplate(
		w,
		"dashboard.html",
		nil,
		"web/templates/navbar.html",
		"web/templates/dashboard.html",
	)
}

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

	renderTemplate(
		w,
		"shops.html",
		data,
		"web/templates/navbar.html",
		"web/templates/shops.html",
	)
}

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

	renderTemplate(
		w,
		"shop.html",
		shop,
		"web/templates/navbar.html",
		"web/templates/shop.html",
	)
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

		renderTemplate(
			w,
			"order.html",
			data,
			"web/templates/navbar.html",
			"web/templates/order.html",
		)

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
			price = shop.Price45kg

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

		if _, err = h.orderService.CreateOrder(order); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/orders", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

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

	renderTemplate(
		w,
		"orders.html",
		data,
		"web/templates/navbar.html",
		"web/templates/orders.html",
	)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := middleware.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
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
		http.Error(
			w,
			"You are not authorized to update this order",
			http.StatusForbidden,
		)
		return
	}

	status := r.FormValue("status")
	if status == "" {
		http.Error(w, "Invalid order status", http.StatusBadRequest)
		return
	}

	if err := h.orderService.UpdateOrderStatus(id, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/shop/orders", http.StatusSeeOther)
}

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

	renderTemplate(
		w,
		"shop_orders.html",
		data,
		"web/templates/navbar.html",
		"web/templates/shop_orders.html",
	)
}

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
		renderTemplate(
			w,
			"shop_manage.html",
			shop,
			"web/templates/shop_manage.html",
		)

	case http.MethodPost:
		price6kg, err := strconv.ParseFloat(r.FormValue("price_6kg"), 64)
		if err != nil || price6kg < 0 {
			http.Error(w, "Invalid 6kg price", http.StatusBadRequest)
			return
		}

		price13kg, err := strconv.ParseFloat(r.FormValue("price_13kg"), 64)
		if err != nil || price13kg < 0 {
			http.Error(w, "Invalid 13kg price", http.StatusBadRequest)
			return
		}

		price45kg, err := strconv.ParseFloat(r.FormValue("price_45kg"), 64)
		if err != nil || price45kg < 0 {
			http.Error(w, "Invalid 45kg price", http.StatusBadRequest)
			return
		}

		err = h.shopService.UpdateShopPrices(
			shop.ID,
			price6kg,
			price13kg,
			price45kg,
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
		http.Error(
			w,
			"You are not authorized to delete this order",
			http.StatusForbidden,
		)
		return
	}

	if err := h.orderService.DeleteOrder(id); err != nil {
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

	renderTemplate(
		w,
		"admin.html",
		data,
		"web/templates/admin.html",
	)
}