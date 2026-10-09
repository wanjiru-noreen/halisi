package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func ConnectDatabase() {
	var err error

	DB, err = sql.Open("sqlite3", "halisi.db")
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
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL,
		email_verified INTEGER NOT NULL DEFAULT 0,
		verification_token TEXT DEFAULT ''
	);
	`

	shopTable := `
	CREATE TABLE IF NOT EXISTS shops (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		location TEXT NOT NULL,
		phone TEXT NOT NULL,
		price_6kg REAL NOT NULL DEFAULT 0,
		price_13kg REAL NOT NULL DEFAULT 0,
		price_45kg REAL NOT NULL DEFAULT 0,
		owner_id INTEGER,
		latitude REAL DEFAULT 0,
		longitude REAL DEFAULT 0,
		business_registration_number TEXT DEFAULT '',
		kra_pin TEXT DEFAULT '',
		license_number TEXT DEFAULT '',
		verification_status TEXT NOT NULL DEFAULT 'pending'
	);
	`

	orderTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		shop_id INTEGER,
		cylinder_size TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		total_price REAL NOT NULL,
		status TEXT NOT NULL,
		delivery_address TEXT NOT NULL DEFAULT ''
	);
	`

	kycDocumentTable := `
	CREATE TABLE IF NOT EXISTS kyc_documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shop_id INTEGER NOT NULL,
		document_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		rejection_reason TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		reviewed_at DATETIME
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

	ensureUserColumns()
	ensureOrderColumns()
	ensureShopColumns()
}

func ensureUserColumns() {
	columns := map[string]string{
		"email_verified":     "INTEGER NOT NULL DEFAULT 0",
		"verification_token": "TEXT DEFAULT ''",
		"google_id": "TEXT NOT NULL DEFAULT ''",
	}

	for column, definition := range columns {
		var count int

		err := DB.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = ?`,
			column,
		).Scan(&count)

		if err != nil {
			log.Fatal(err)
		}

		if count == 0 {
			_, err := DB.Exec(
				"ALTER TABLE users ADD COLUMN " + column + " " + definition,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func ensureOrderColumns() {
	columns := map[string]string{
		"delivery_address": "TEXT NOT NULL DEFAULT ''",
	}

	for column, definition := range columns {
		var count int

		err := DB.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('orders') WHERE name = ?`,
			column,
		).Scan(&count)

		if err != nil {
			log.Fatal(err)
		}

		if count == 0 {
			_, err := DB.Exec(
				"ALTER TABLE orders ADD COLUMN " + column + " " + definition,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func ensureShopColumns() {
	columns := map[string]string{
		"latitude":                     "REAL DEFAULT 0",
		"longitude":                    "REAL DEFAULT 0",
		"business_registration_number": "TEXT DEFAULT ''",
		"kra_pin":                      "TEXT DEFAULT ''",
		"license_number":               "TEXT DEFAULT ''",
		"verification_status":          "TEXT NOT NULL DEFAULT 'pending'",
	}

	for column, definition := range columns {
		var count int

		err := DB.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('shops') WHERE name = ?`,
			column,
		).Scan(&count)

		if err != nil {
			log.Fatal(err)
		}

		if count == 0 {
			_, err := DB.Exec(
				"ALTER TABLE shops ADD COLUMN " + column + " " + definition,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}