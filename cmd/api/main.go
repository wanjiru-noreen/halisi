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

	database.ConnectDatabase()

	userRepository := repository.NewUserRepository()

	userService := services.NewUserService(userRepository)
	shopService := services.NewShopService(userRepository)

	orderRepository := repository.NewOrderRepository()
	orderService := services.NewOrderService(orderRepository)

	handler := handlers.NewHandler(
		userService,
		shopService,
		orderService,
	)

	http.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("web/static"))))

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
