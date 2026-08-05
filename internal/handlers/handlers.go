package handlers

import (
	"halisi/internal/models"
	"halisi/internal/services"
	"html/template"
	"net/http"
)

type Handler struct {
	userService *services.UserService
}

func NewHandler(service *services.UserService) *Handler {
	return &Handler{
		userService: service,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	user := models.User{
		Name:     "Test User",
		Email:    "test@halisi.com",
		Password: "password",
		Role:     "customer",
	}

	createdUser := h.userService.RegisterUser(user)

	w.Write([]byte("User created: " + createdUser.Name))
}
