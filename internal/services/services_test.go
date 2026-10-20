package services

import (
	"database/sql"
	"testing"

	"halisi/internal/database"
	"halisi/internal/models"
	"halisi/internal/repository"

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
			role TEXT NOT NULL,
			email_verified INTEGER NOT NULL DEFAULT 0,
			verification_token TEXT DEFAULT '',
			google_id TEXT NOT NULL DEFAULT ''
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

func TestRegisterGoogleUser(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	userRepo := repository.NewUserRepository()
	service := NewUserService(userRepo)

	t.Run("NewGoogleUser", func(t *testing.T) {
		user, err := service.RegisterGoogleUser("Alice", "alice@example.com", "google-sub-1")
		if err != nil {
			t.Fatalf("RegisterGoogleUser() error = %v", err)
		}

		if user.ID == 0 {
			t.Fatal("expected assigned user ID")
		}
		if user.Email != "alice@example.com" {
			t.Errorf("expected alice@example.com, got %q", user.Email)
		}
		if !user.EmailVerified {
			t.Error("expected Google user email to be verified")
		}
		if user.Role != "customer" {
			t.Errorf("expected customer role, got %q", user.Role)
		}
		if user.GoogleID != "google-sub-1" {
			t.Errorf("expected google-sub-1, got %q", user.GoogleID)
		}
	})

	t.Run("ReturningGoogleUser", func(t *testing.T) {
		user, err := service.RegisterGoogleUser("Alice Updated", "alice@example.com", "google-sub-1")
		if err != nil {
			t.Fatalf("RegisterGoogleUser() returning user error = %v", err)
		}

		if user.GoogleID != "google-sub-1" {
			t.Errorf("expected google-sub-1, got %q", user.GoogleID)
		}
	})

	t.Run("LinkExistingVerifiedUser", func(t *testing.T) {
		// Existing email/password user who verified their email
		created, err := userRepo.CreateUser(models.User{
			Name:          "Bob",
			Email:         "bob@example.com",
			Password:      "hashed_pw",
			Role:          "customer",
			EmailVerified: true,
			GoogleID:      "",
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}

		user, err := service.RegisterGoogleUser("Bob", "bob@example.com", "google-sub-bob")
		if err != nil {
			t.Fatalf("RegisterGoogleUser() linking error = %v", err)
		}

		if user.ID != created.ID {
			t.Errorf("expected user ID %d, got %d", created.ID, user.ID)
		}
		if user.GoogleID != "google-sub-bob" {
			t.Errorf("expected google-sub-bob, got %q", user.GoogleID)
		}

		// Verify database was updated
		inDB, err := userRepo.GetUserByEmail("bob@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v", err)
		}
		if inDB.GoogleID != "google-sub-bob" {
			t.Errorf("expected database to have google-sub-bob, got %q", inDB.GoogleID)
		}
	})

	t.Run("ExistingUnverifiedUserFails", func(t *testing.T) {
		_, err := userRepo.CreateUser(models.User{
			Name:          "Charlie",
			Email:         "charlie@example.com",
			Password:      "hashed_pw",
			Role:          "customer",
			EmailVerified: false,
			GoogleID:      "",
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}

		_, err = service.RegisterGoogleUser("Charlie", "charlie@example.com", "google-sub-charlie")
		if err == nil {
			t.Fatal("expected error linking unverified user with Google")
		}
	})

	t.Run("AlreadyLinkedToDifferentGoogleAccount", func(t *testing.T) {
		// Bob is already linked to google-sub-bob
		_, err := service.RegisterGoogleUser("Bob", "bob@example.com", "google-sub-bob-other")
		if err == nil {
			t.Fatal("expected error linking account already linked to a different Google account")
		}
	})

	t.Run("GoogleIDAlreadyLinkedToAnotherUser", func(t *testing.T) {
		// Create verified Dan
		_, err := userRepo.CreateUser(models.User{
			Name:          "Dan",
			Email:         "dan@example.com",
			Password:      "hashed_pw",
			Role:          "customer",
			EmailVerified: true,
			GoogleID:      "",
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}

		// Try to link Dan with Alice's Google ID (google-sub-1)
		_, err = service.RegisterGoogleUser("Dan", "dan@example.com", "google-sub-1")
		if err == nil {
			t.Fatal("expected error linking Google ID already belonging to another user")
		}
	})

	t.Run("InvalidInputs", func(t *testing.T) {
		if _, err := service.RegisterGoogleUser("Name", "", "sub-1"); err == nil {
			t.Error("expected error for empty email")
		}
		if _, err := service.RegisterGoogleUser("Name", "email@example.com", ""); err == nil {
			t.Error("expected error for empty googleID")
		}
		if _, err := service.RegisterGoogleUser("Name", "invalid-email", "sub-1"); err == nil {
			t.Error("expected error for invalid email format")
		}
	})
}
