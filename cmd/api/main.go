package main

import (
	"fmt"
	"halisi/internal/database"
	"halisi/internal/handlers"
	"halisi/internal/repository"
	"halisi/internal/services"
	"net/http"
)

func main() {

	// Connect to SQLite database
	database.ConnectDatabase()

	// Initialize application layers
	userRepository := repository.NewUserRepository()

	userService := services.NewUserService(userRepository)
	shopService := services.NewShopService(userRepository)

	handler := handlers.NewHandler(
		userService,
		shopService,
	)

	// Serve static files
	http.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("web/static"))))

	// Routes
	http.HandleFunc("/", handler.Home)
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/login", handler.Login)
	http.HandleFunc("/dashboard", handler.Dashboard)
	http.HandleFunc("/shops", handler.Shops)
	http.HandleFunc("/shop", handler.Shop)
	http.HandleFunc("/order", handler.Order)
	http.HandleFunc("/orders", handler.Orders)

	fmt.Println("Halisi server running on http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
