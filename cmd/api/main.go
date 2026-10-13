package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"halisi/internal/database"
	"halisi/internal/handlers"
	"halisi/internal/middleware"
	"halisi/internal/repository"
	"halisi/internal/services"
)

func main() {
	// Load environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Validate Google OAuth credentials.
	if os.Getenv("GOOGLE_CLIENT_ID") == "" ||
		os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		log.Fatal("Google OAuth credentials are not configured")
	}

	// Initialize Google OAuth and OpenID Connect.
	if err := handlers.ConfigureGoogleAuth(); err != nil {
		log.Fatalf("Failed to configure Google authentication: %v", err)
	}

	// Connect to database.
	database.ConnectDatabase()

	// Create repositories.
	userRepository := repository.NewUserRepository()
	shopRepository := repository.NewShopRepository()
	orderRepository := repository.NewOrderRepository()
	kycDocumentRepository := repository.NewKYCDocumentRepository()

	// Create services.
	userService := services.NewUserService(userRepository)
	shopService := services.NewShopService(shopRepository)
	orderService := services.NewOrderService(orderRepository)
	kycDocumentService := services.NewKYCDocumentService(
		kycDocumentRepository,
	)

	// Create handler.
	handler := handlers.NewHandler(
		userService,
		shopService,
		orderService,
		kycDocumentService,
	)

	// Serve static files.
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle(
		"/static/",
		http.StripPrefix("/static/", fs),
	)

	// Public routes.
	http.HandleFunc("/", handler.Home)
	http.HandleFunc("/login", handler.Login)
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/verify-email", handler.VerifyEmail)
	http.HandleFunc("/logout", handler.Logout)

	// Google authentication routes.
	http.HandleFunc("/auth/google", handler.GoogleLogin)
	http.HandleFunc("/auth/google/callback", handler.GoogleCallback)

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

	http.HandleFunc(
		"/orders/confirmation",
		middleware.RequireRole(
			"customer",
			handler.OrderConfirmation,
		),
	)

	// ---------------- ADMIN ROUTES ----------------

	http.HandleFunc(
		"/admin",
		middleware.RequireRole("admin", handler.AdminDashboard),
	)

	http.HandleFunc(
		"/admin/verify-shop",
		middleware.RequireRole("admin", handler.AdminVerifyShop),
	)

	http.HandleFunc(
		"/admin/kyc",
		middleware.RequireRole("admin", handler.GetPendingKYCDocuments),
	)

	http.HandleFunc(
		"/admin/kyc/review",
		middleware.RequireRole("admin", handler.ReviewKYCDocument),
	)

	http.HandleFunc(
		"/admin/kyc/document",
		middleware.RequireRole("admin", handler.ViewKYCDocument),
	)

	// ---------------- SHOP OWNER ROUTES ----------------

	http.HandleFunc(
		"/shop/orders",
		middleware.RequireRole("owner", handler.ShopOrders),
	)

	http.HandleFunc(
		"/shop/manage",
		middleware.RequireRole("owner", handler.ShopManage),
	)

	http.HandleFunc(
		"/shop/kyc",
		middleware.RequireRole("owner", handler.GetKYCDocuments),
	)

	http.HandleFunc(
		"/shop/kyc/upload",
		middleware.RequireRole("owner", handler.UploadKYCDocument),
	)

	http.HandleFunc(
		"/orders/update",
		middleware.RequireRole("owner", handler.UpdateOrderStatus),
	)

	// Start server.
	log.Println("🚀 Halisi server running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
