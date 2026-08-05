package handlers

import (
	"html/template"
	"net/http"

	"halisi/internal/models"
	"halisi/internal/services"
)

type Handler struct {
	userService *services.UserService
}

func NewHandler(service *services.UserService) *Handler {
	return &Handler{
		userService: service,
	}
}

// Home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Register page
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/register.html"))

		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:

		user := models.User{
			Name:     r.FormValue("name"),
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
			Role:     "customer",
		}

		_, err := h.userService.RegisterUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Login page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/login.html"))

		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := h.userService.LoginUser(email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if user.ID == 0 {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		if user.Password != password {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/dashboard.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Shops page
func (h *Handler) Shops(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/shops.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Shop page
func (h *Handler) Shop(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/shop.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Order page
func (h *Handler) Order(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("web/templates/order.html"))

		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		http.Redirect(w, r, "/orders", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Orders page
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/orders.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
