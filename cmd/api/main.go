package main

import (
	"fmt"
	"halisi/internal/handlers"
	"halisi/internal/repository"
	"halisi/internal/services"
	"net/http"
)

func main() {

	userRepository := repository.NewUserRepository()

	userService := services.NewUserService(userRepository)

	handler := handlers.NewHandler(userService)

	http.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("web/static"))))

	http.HandleFunc("/", handler.Home)
	http.HandleFunc("/register", handler.Register)

	fmt.Println("Halisi server running on port 8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
