package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"halisi/internal/middleware"
	"halisi/internal/models"
	"halisi/internal/services"

	"github.com/gorilla/sessions"
)

type Handler struct {
	OrderService *services.OrderService
	ShopService  *services.ShopService
	UserService  *services.UserService
	Store        *sessions.CookieStore
}

func NewHandler(userService *services.UserService, shopService *services.ShopService, orderService *services.OrderService) *Handler {
	return &Handler{
		OrderService: orderService,
		ShopService:  shopService,
		UserService:  userService,
		Store:        middleware.Store,
	}
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}, files ...string) {
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Failed to load page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, name, data); err != nil {
		http.Error(w, "Failed to render page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(output.Bytes())
}

// Register handles user registration form submission
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, "register.html", nil, "web/templates/register.html")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fullName := r.FormValue("fullname")
	if fullName == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	if password == "" {
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	role := r.FormValue("account_type")
	if role == "vendor" {
		role = "owner"
	}

	if role != "customer" && role != "owner" {
		http.Error(w, "Invalid account type", http.StatusBadRequest)
		return
	}

	shopName := r.FormValue("shop_name")
	location := r.FormValue("location")
	phone := r.FormValue("phone")

	if role == "owner" && (shopName == "" || location == "" || phone == "") {
		http.Error(w, "Shop name, location, and phone are required", http.StatusBadRequest)
		return
	}

	user := models.User{
		Name:     fullName,
		Email:    email,
		Password: password,
		Role:     role,
	}

	registeredUser, err := h.UserService.RegisterUser(user)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	if role == "owner" {
		ownerID := registeredUser.ID

		if _, err := h.ShopService.CreateShop(models.Shop{
			Name:     shopName,
			Location: location,
			Phone:    phone,
			OwnerID:  &ownerID,
		}); err != nil {
			log.Printf("Error creating shop: %v", err)
			http.Error(w, "Failed to create shop", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RenderOrderPage handles displaying the order form with available shops
func (h *Handler) RenderOrderPage(w http.ResponseWriter, r *http.Request) {
	shops, err := h.ShopService.GetShops()
	if err != nil {
		log.Printf("Error fetching shops: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	selectedShopIDStr := r.URL.Query().Get("shop_id")

	var selectedShopID int

	if selectedShopIDStr != "" {
		selectedShopID, _ = strconv.Atoi(selectedShopIDStr)
	}

	data := struct {
		Shops          []models.Shop
		SelectedShopID int
	}{
		Shops:          shops,
		SelectedShopID: selectedShopID,
	}

	renderTemplate(
		w,
		"order.html",
		data,
		"web/templates/navbar.html",
		"web/templates/order.html",
	)
}

// CreateOrder handles POST requests from the order form
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/order", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	shopID, err := strconv.Atoi(r.FormValue("shop_id"))
	if err != nil || shopID <= 0 {
		http.Error(w, "Invalid shop", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil || quantity <= 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	cylinderSize := r.FormValue("cylinder_size")
	address := r.FormValue("address")

	shop, err := h.ShopService.GetShopByID(shopID)
	if err != nil || shop.ID == 0 {
		http.Error(w, "Shop not found", http.StatusBadRequest)
		return
	}

	price := map[string]float64{
		"6kg":  shop.Price6kg,
		"13kg": shop.Price13kg,
		"45kg": shop.Price45kg,
	}[cylinderSize]

	if price <= 0 {
		http.Error(w, "Invalid cylinder size or unavailable price", http.StatusBadRequest)
		return
	}

	session, _ := h.Store.Get(r, "halisi-session")

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	order := &models.Order{
		UserID:          userID,
		ShopID:          shopID,
		CylinderSize:    cylinderSize,
		Quantity:        quantity,
		DeliveryAddress: address,
		TotalPrice:      price * float64(quantity),
		Status:          "Pending",
	}

	createdOrder, err := h.OrderService.CreateOrder(*order)
	if err != nil {
		log.Printf("Error creating order: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/orders/confirmation?id="+strconv.Itoa(createdOrder.ID),
		http.StatusSeeOther,
	)
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	renderTemplate(
		w,
		"index.html",
		nil,
		"web/templates/index.html",
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(
			w,
			"login.html",
			nil,
			"web/templates/login.html",
		)
		return
	}

	user, err := h.UserService.LoginUser(
		r.FormValue("email"),
		r.FormValue("password"),
	)

	if err != nil || user.ID == 0 {
		http.Error(
			w,
			"Invalid email or password",
			http.StatusUnauthorized,
		)
		return
	}

	session, err := h.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(
			w,
			"Failed to get session",
			http.StatusInternalServerError,
		)
		return
	}

	session.Values["user_id"] = user.ID
	session.Values["role"] = user.Role

	if err := session.Save(r, w); err != nil {
		http.Error(
			w,
			"Failed to save session",
			http.StatusInternalServerError,
		)
		return
	}

	if user.Role == "owner" {
		http.Redirect(
			w,
			r,
			"/shop/orders",
			http.StatusSeeOther,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/dashboard",
		http.StatusSeeOther,
	)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	renderTemplate(
		w,
		"dashboard.html",
		nil,
		"web/templates/dashboard.html",
	)
}

func (h *Handler) Shops(w http.ResponseWriter, r *http.Request) {
	shops, err := h.ShopService.GetShops()
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	renderTemplate(
		w,
		"shops.html",
		struct {
			Shops []models.Shop
		}{
			Shops: shops,
		},
		"web/templates/navbar.html",
		"web/templates/shops.html",
	)
}

func (h *Handler) Shop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(
			w,
			"Invalid shop ID",
			http.StatusBadRequest,
		)
		return
	}

	shop, err := h.ShopService.GetShopByID(id)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
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
	if r.Method == http.MethodPost {
		h.CreateOrder(w, r)
		return
	}

	h.RenderOrderPage(w, r)
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	session, _ := h.Store.Get(r, "halisi-session")

	userID, _ := session.Values["user_id"].(int)

	orders, err := h.OrderService.GetOrdersByUserID(userID)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	renderTemplate(
		w,
		"orders.html",
		struct {
			Orders []models.Order
		}{
			Orders: orders,
		},
		"web/templates/navbar.html",
		"web/templates/orders.html",
	)
}

func (h *Handler) OrderConfirmation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(
			w,
			"Invalid order ID",
			http.StatusBadRequest,
		)
		return
	}

	order, err := h.OrderService.GetOrderByID(id)
	if err != nil {
		http.Error(
			w,
			"Order not found",
			http.StatusNotFound,
		)
		return
	}

	renderTemplate(
		w,
		"order_confirmation.html",
		order,
		"web/templates/navbar.html",
		"web/templates/order_confirmation.html",
	)
}

func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(
			w,
			"Invalid order ID",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.OrderService.DeleteOrder(id); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/orders",
		http.StatusSeeOther,
	)
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	renderTemplate(
		w,
		"admin.html",
		nil,
		"web/templates/navbar.html",
		"web/templates/admin.html",
	)
}

func (h *Handler) ShopOrders(w http.ResponseWriter, r *http.Request) {
	session, err := h.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(
			w,
			"Failed to get session",
			http.StatusInternalServerError,
		)
		return
	}

	ownerID, ok := session.Values["user_id"].(int)
	if !ok || ownerID == 0 {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	shop, err := h.ShopService.GetShopByOwnerID(ownerID)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	orders, err := h.OrderService.GetOrdersByShopID(shop.ID)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
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

func (h *Handler) ShopManage(w http.ResponseWriter, r *http.Request) {
	session, err := h.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(
			w,
			"Failed to get session",
			http.StatusInternalServerError,
		)
		return
	}

	ownerID, ok := session.Values["user_id"].(int)
	if !ok || ownerID == 0 {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	shop, err := h.ShopService.GetShopByOwnerID(ownerID)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	if shop.ID == 0 {
		http.Error(
			w,
			"Shop not found for this account",
			http.StatusNotFound,
		)
		return
	}

	if r.Method == http.MethodPost {
		price6kg, err6 := strconv.ParseFloat(
			r.FormValue("price_6kg"),
			64,
		)

		price13kg, err13 := strconv.ParseFloat(
			r.FormValue("price_13kg"),
			64,
		)

		price45kg, err45 := strconv.ParseFloat(
			r.FormValue("price_45kg"),
			64,
		)

		if err6 != nil ||
			err13 != nil ||
			err45 != nil ||
			price6kg < 0 ||
			price13kg < 0 ||
			price45kg < 0 {
			http.Error(
				w,
				"Invalid prices",
				http.StatusBadRequest,
			)
			return
		}

		if err := h.ShopService.UpdateShopPrices(
			shop.ID,
			price6kg,
			price13kg,
			price45kg,
		); err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/shop/manage",
			http.StatusSeeOther,
		)
		return
	}

	renderTemplate(
		w,
		"shop_manage.html",
		shop,
		"web/templates/navbar.html",
		"web/templates/shop_manage.html",
	)
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(
			w,
			"Invalid order ID",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.OrderService.UpdateOrderStatus(
		id,
		r.FormValue("status"),
	); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/shop/orders",
		http.StatusSeeOther,
	)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.Store.Get(r, "halisi-session")

	if err == nil {
		session.Options.MaxAge = -1
		_ = session.Save(r, w)
	}

	http.Redirect(
		w,
		r,
		"/login",
		http.StatusSeeOther,
	)
}
