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

	// ---------------- CUSTOMER ROUTES ----------------

	http.HandleFunc(
		"/dashboard",
		middleware.RequireRole("customer", handler.Dashboard),
	)

	http.HandleFunc(
		"/shops",
		middleware.RequireRole("customer", handler.Shops),
	)

	http.HandleFunc(
		"/shop",
		middleware.RequireRole("customer", handler.Shop),
	)

	http.HandleFunc(
		"/order",
		middleware.RequireRole("customer", handler.Order),
	)

	http.HandleFunc(
		"/orders",
		middleware.RequireRole("customer", handler.Orders),
	)

	http.HandleFunc(
		"/orders/delete",
		middleware.RequireRole("customer", handler.DeleteOrder),
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
