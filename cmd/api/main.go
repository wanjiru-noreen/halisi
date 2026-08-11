package main

import (
	"log"
	"net/http"

	"halisi/internal/database"
	"halisi/internal/handlers"
	"halisi/internal/middleware"
	"halisi/internal/repository"
	"halisi/internal/services"
)

func main() {
	// Connect to database
	database.ConnectDatabase()

	// Create repositories
	userRepository := repository.NewUserRepository()
	shopRepository := repository.NewShopRepository()
	orderRepository := repository.NewOrderRepository()

	// Create services
	userService := services.NewUserService(userRepository)
	shopService := services.NewShopService(shopRepository)
	orderService := services.NewOrderService(orderRepository)

	// Create handler
	handler := handlers.NewHandler(
		userService,
		shopService,
		orderService,
	)

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// ---------------- PUBLIC ROUTES ----------------

	http.HandleFunc("/", handler.Home)
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/login", handler.Login)

	// ---------------- CUSTOMER ROUTES ----------------

	http.HandleFunc(
		"/dashboard",
		middleware.RequireAuth(handler.Dashboard),
	)

	http.HandleFunc(
		"/shops",
		middleware.RequireAuth(handler.Shops),
	)

	http.HandleFunc(
		"/shop",
		middleware.RequireAuth(handler.Shop),
	)

	http.HandleFunc(
		"/order",
		middleware.RequireAuth(handler.Order),
	)

	http.HandleFunc(
		"/orders",
		middleware.RequireAuth(handler.Orders),
	)

	http.HandleFunc(
		"/orders/delete",
		middleware.RequireAuth(handler.DeleteOrder),
	)

	http.HandleFunc(
		"/logout",
		middleware.RequireAuth(handler.Logout),
	)

	// ---------------- SHOP OWNER ROUTES ----------------

	http.HandleFunc(
		"/shop/orders",
		middleware.RequireRole("owner", handler.ShopOrders),
	)

	http.HandleFunc(
		"/shop/manage",
		middleware.RequireRole("owner", handler.ManageShop),
	)

	http.HandleFunc(
		"/orders/update",
		middleware.RequireRole("owner", handler.UpdateOrderStatus),
	)

	// ---------------- ADMIN ROUTES ----------------

	http.HandleFunc(
		"/admin",
		middleware.RequireRole("admin", handler.AdminDashboard),
	)

	// Start server
	log.Println("🚀 Halisi server running on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
