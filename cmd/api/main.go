package main

import (
	"fmt"
	"net/http"

	"halisi/internal/database"
	"halisi/internal/handlers"
	"halisi/internal/middleware"
	"halisi/internal/repository"
	"halisi/internal/services"
)

func main() {

	// Connect database
	database.ConnectDatabase()

	// Initialize repositories
	userRepository := repository.NewUserRepository()
	orderRepository := repository.NewOrderRepository()

	// Initialize services
	userService := services.NewUserService(userRepository)
	shopService := services.NewShopService(userRepository)
	orderService := services.NewOrderService(orderRepository)

	// Initialize handlers
	handler := handlers.NewHandler(
		userService,
		shopService,
		orderService,
	)

	// Static files
	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("web/static")),
		),
	)

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
		"/orders/update",
		middleware.RequireRole("owner", handler.UpdateOrderStatus),
	)

	// ---------------- ADMIN ROUTES ----------------

	http.HandleFunc(
		"/admin",
		middleware.RequireRole("admin", handler.AdminDashboard),
	)

	fmt.Println("🚀 Halisi server running on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
