package repository

import (
	"database/sql"
	"testing"

	"halisi/internal/database"
	"halisi/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	var err error

	database.DB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = database.DB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			role TEXT NOT NULL
		);

		CREATE TABLE shops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			location TEXT NOT NULL,
			phone TEXT NOT NULL,
			price_6kg REAL NOT NULL DEFAULT 0,
			price_13kg REAL NOT NULL DEFAULT 0,
			price_45kg REAL NOT NULL DEFAULT 0,
			owner_id INTEGER
		);

		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			shop_id INTEGER,
			cylinder_size TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			total_price REAL NOT NULL,
			status TEXT NOT NULL,
			delivery_address TEXT NOT NULL DEFAULT ''
		);
	`)

	if err != nil {
		t.Fatal(err)
	}
}

func teardownTestDB() {
	if database.DB != nil {
		database.DB.Close()
		database.DB = nil
	}
}

/* =========================
   USER REPOSITORY
========================= */

func TestUserRepository(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	repo := NewUserRepository()

	t.Run("CreateUser", func(t *testing.T) {

		user := models.User{
			Name:     "John Doe",
			Email:    "john@example.com",
			Password: "hashed-password",
			Role:     "customer",
		}

		created, err := repo.CreateUser(user)
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}

		if created.ID == 0 {
			t.Fatal("expected user ID to be assigned")
		}

		if created.Name != user.Name {
			t.Errorf("expected name %q, got %q",
				user.Name,
				created.Name,
			)
		}

		if created.Email != user.Email {
			t.Errorf("expected email %q, got %q",
				user.Email,
				created.Email,
			)
		}
	})

	t.Run("GetUserByEmail", func(t *testing.T) {

		user, err := repo.GetUserByEmail("john@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v", err)
		}

		if user.ID == 0 {
			t.Fatal("expected user to be found")
		}

		if user.Name != "John Doe" {
			t.Errorf("expected John Doe, got %q", user.Name)
		}
	})

	t.Run("GetUserByEmailNotFound", func(t *testing.T) {

		user, err := repo.GetUserByEmail("missing@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v", err)
		}

		if user.ID != 0 {
			t.Errorf("expected empty user, got ID %d", user.ID)
		}
	})

	t.Run("GetUsers", func(t *testing.T) {

		_, err := repo.CreateUser(models.User{
			Name:     "Jane Doe",
			Email:    "jane@example.com",
			Password: "password",
			Role:     "owner",
		})

		if err != nil {
			t.Fatal(err)
		}

		users, err := repo.GetUsers()
		if err != nil {
			t.Fatalf("GetUsers() error = %v", err)
		}

		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d", len(users))
		}
	})
}

/* =========================
   SHOP REPOSITORY
========================= */

func TestShopRepository(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	repo := NewShopRepository()

	ownerID := 10

	t.Run("CreateShop", func(t *testing.T) {

		shop := models.Shop{
			Name:      "Test Gas",
			Location:  "Kisumu CBD",
			Phone:     "0712345678",
			Price6kg:  1400,
			Price13kg: 2800,
			Price45kg: 12000,
			OwnerID:   &ownerID,
		}

		created, err := repo.CreateShop(shop)
		if err != nil {
			t.Fatalf("CreateShop() error = %v", err)
		}

		if created.ID == 0 {
			t.Fatal("expected shop ID to be assigned")
		}

		if created.Name != "Test Gas" {
			t.Errorf("expected Test Gas, got %q", created.Name)
		}
	})

	t.Run("GetShopByID", func(t *testing.T) {

		shop, err := repo.GetShopByID(1)
		if err != nil {
			t.Fatalf("GetShopByID() error = %v", err)
		}

		if shop.ID != 1 {
			t.Errorf("expected ID 1, got %d", shop.ID)
		}

		if shop.Price6kg != 1400 {
			t.Errorf("expected price 1400, got %f", shop.Price6kg)
		}
	})

	t.Run("GetShopByIDNotFound", func(t *testing.T) {

		shop, err := repo.GetShopByID(999)
		if err != nil {
			t.Fatalf("GetShopByID() error = %v", err)
		}

		if shop.ID != 0 {
			t.Errorf("expected empty shop, got ID %d", shop.ID)
		}
	})

	t.Run("GetShopByOwnerID", func(t *testing.T) {

		shop, err := repo.GetShopByOwnerID(ownerID)
		if err != nil {
			t.Fatalf("GetShopByOwnerID() error = %v", err)
		}

		if shop.ID == 0 {
			t.Fatal("expected shop to be found")
		}

		if shop.OwnerID == nil {
			t.Fatal("expected owner ID")
		}

		if *shop.OwnerID != ownerID {
			t.Errorf(
				"expected owner ID %d, got %d",
				ownerID,
				*shop.OwnerID,
			)
		}
	})

	t.Run("UpdateShopPrices", func(t *testing.T) {

		err := repo.UpdateShopPrices(
			1,
			1500,
			3000,
			12500,
		)

		if err != nil {
			t.Fatalf("UpdateShopPrices() error = %v", err)
		}

		shop, err := repo.GetShopByID(1)
		if err != nil {
			t.Fatal(err)
		}

		if shop.Price6kg != 1500 {
			t.Errorf("expected 1500, got %f", shop.Price6kg)
		}

		if shop.Price13kg != 3000 {
			t.Errorf("expected 3000, got %f", shop.Price13kg)
		}

		if shop.Price45kg != 12500 {
			t.Errorf("expected 12500, got %f", shop.Price45kg)
		}
	})

	t.Run("GetShops", func(t *testing.T) {

		shops, err := repo.GetShops()
		if err != nil {
			t.Fatalf("GetShops() error = %v", err)
		}

		if len(shops) != 1 {
			t.Fatalf("expected 1 shop, got %d", len(shops))
		}
	})
}

/* =========================
   ORDER REPOSITORY
========================= */

func TestOrderRepository(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	repo := NewOrderRepository()

	_, err := database.DB.Exec(`
		INSERT INTO shops (
			name,
			location,
			phone,
			price_6kg,
			price_13kg,
			price_45kg
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		"Shell Gas",
		"Kisumu CBD",
		"0712345678",
		1400,
		2800,
		12000,
	)

	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateOrder", func(t *testing.T) {

		order := models.Order{
			UserID:          1,
			ShopID:          1,
			CylinderSize:    "6kg",
			Quantity:        2,
			TotalPrice:      2800,
			Status:          "Pending",
			DeliveryAddress: "Kisumu CBD",
		}

		created, err := repo.CreateOrder(order)
		if err != nil {
			t.Fatalf("CreateOrder() error = %v", err)
		}

		if created.ID == 0 {
			t.Fatal("expected order ID to be assigned")
		}

		if created.Quantity != 2 {
			t.Errorf("expected quantity 2, got %d", created.Quantity)
		}
	})

	t.Run("GetOrderByID", func(t *testing.T) {

		order, err := repo.GetOrderByID(1)
		if err != nil {
			t.Fatalf("GetOrderByID() error = %v", err)
		}

		if order.ID != 1 {
			t.Errorf("expected ID 1, got %d", order.ID)
		}

		if order.ShopName != "Shell Gas" {
			t.Errorf(
				"expected Shell Gas, got %q",
				order.ShopName,
			)
		}
	})

	t.Run("GetOrders", func(t *testing.T) {

		orders, err := repo.GetOrders()
		if err != nil {
			t.Fatalf("GetOrders() error = %v", err)
		}

		if len(orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(orders))
		}
	})

	t.Run("GetOrdersByUserID", func(t *testing.T) {

		orders, err := repo.GetOrdersByUserID(1)
		if err != nil {
			t.Fatalf(
				"GetOrdersByUserID() error = %v",
				err,
			)
		}

		if len(orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(orders))
		}
	})

	t.Run("GetOrdersByShopID", func(t *testing.T) {

		orders, err := repo.GetOrdersByShopID(1)
		if err != nil {
			t.Fatalf(
				"GetOrdersByShopID() error = %v",
				err,
			)
		}

		if len(orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(orders))
		}
	})

	t.Run("UpdateOrderStatus", func(t *testing.T) {

		err := repo.UpdateOrderStatus(1, "Accepted")
		if err != nil {
			t.Fatalf(
				"UpdateOrderStatus() error = %v",
				err,
			)
		}

		order, err := repo.GetOrderByID(1)
		if err != nil {
			t.Fatal(err)
		}

		if order.Status != "Accepted" {
			t.Errorf(
				"expected Accepted, got %q",
				order.Status,
			)
		}
	})

	t.Run("DeletePendingOrder", func(t *testing.T) {

		_, err := repo.CreateOrder(models.Order{
			UserID:          2,
			ShopID:          1,
			CylinderSize:    "13kg",
			Quantity:        1,
			TotalPrice:      2800,
			Status:          "Pending",
			DeliveryAddress: "Kondele",
		})

		if err != nil {
			t.Fatal(err)
		}

		err = repo.DeleteOrder(2)
		if err != nil {
			t.Fatalf(
				"DeleteOrder() error = %v",
				err,
			)
		}

		_, err = repo.GetOrderByID(2)

		if err == nil {
			t.Fatal("expected deleted order to be unavailable")
		}
	})
}
