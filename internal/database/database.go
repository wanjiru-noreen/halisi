package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDatabase() {
	var err error

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTables()

	log.Println("Database connected successfully")
}

func createTables() {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL,
		email_verified BOOLEAN NOT NULL DEFAULT FALSE,
		verification_token TEXT DEFAULT '',
		google_id TEXT NOT NULL DEFAULT ''
	);
	`

	shopTable := `
	CREATE TABLE IF NOT EXISTS shops (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		location TEXT NOT NULL,
		phone TEXT NOT NULL,
		price_6kg DOUBLE PRECISION NOT NULL DEFAULT 0,
		price_13kg DOUBLE PRECISION NOT NULL DEFAULT 0,
		price_45kg DOUBLE PRECISION NOT NULL DEFAULT 0,
		owner_id INTEGER,
		latitude DOUBLE PRECISION DEFAULT 0,
		longitude DOUBLE PRECISION DEFAULT 0,
		business_registration_number TEXT DEFAULT '',
		kra_pin TEXT DEFAULT '',
		license_number TEXT DEFAULT '',
		verification_status TEXT NOT NULL DEFAULT 'pending'
	);
	`

	orderTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER,
		shop_id INTEGER,
		cylinder_size TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		total_price DOUBLE PRECISION NOT NULL,
		status TEXT NOT NULL,
		delivery_address TEXT NOT NULL DEFAULT ''
	);
	`

	kycDocumentTable := `
	CREATE TABLE IF NOT EXISTS kyc_documents (
		id SERIAL PRIMARY KEY,
		shop_id INTEGER NOT NULL,
		document_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		rejection_reason TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		reviewed_at TIMESTAMP
	);
	`

	_, err := DB.Exec(userTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(shopTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(orderTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(kycDocumentTable)
	if err != nil {
		log.Fatal(err)
	}
}
