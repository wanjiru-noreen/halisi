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
	seedShops()

	log.Println("Database connected successfully")
}

func createTables() {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL
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
		owner_id INTEGER
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

	ensureOrderColumns()
	ensureShopColumns()
}

func ensureOrderColumns() {
	rows, err := DB.Query("PRAGMA table_info(orders)")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	columns := map[string]bool{}

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(
			&cid,
			&name,
			&ctype,
			&notnull,
			&dfltValue,
			&pk,
		); err != nil {
			log.Fatal(err)
		}

		columns[name] = true
	}

	if !columns["delivery_address"] {
		_, err := DB.Exec(`
			ALTER TABLE orders
			ADD COLUMN delivery_address TEXT NOT NULL DEFAULT ''
		`)

		if err != nil {
			log.Fatal(err)
		}
	}
}

func ensureShopColumns() {
	rows, err := DB.Query("PRAGMA table_info(shops)")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	columns := map[string]bool{}

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(
			&cid,
			&name,
			&ctype,
			&notnull,
			&dfltValue,
			&pk,
		); err != nil {
			log.Fatal(err)
		}

		columns[name] = true
	}

	if !columns["price_6kg"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN price_6kg REAL NOT NULL DEFAULT 0
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["price_13kg"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN price_13kg REAL NOT NULL DEFAULT 0
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["price_45kg"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN price_45kg REAL NOT NULL DEFAULT 0
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["owner_id"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN owner_id INTEGER
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	// Set default 45kg prices for existing shops
	// that were created before the 45kg column existed.
	_, err = DB.Exec(`
		UPDATE shops
		SET price_45kg = 12000
		WHERE price_45kg = 0
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func seedShops() {
	shops := []struct {
		name     string
		location string
		phone    string
		price6   float64
		price13  float64
		price45  float64
	}{
		{
			"Shell Gas",
			"Kisumu CBD",
			"0712345678",
			1400,
			2800,
			12000,
		},
		{
			"Rubis Gas",
			"Kondele",
			"0723456789",
			1450,
			2900,
			12000,
		},
		{
			"TotalEnergies",
			"Milimani",
			"0734567890",
			1500,
			3000,
			12000,
		},
	}

	for _, shop := range shops {
		var count int

		err := DB.QueryRow(
			"SELECT COUNT(*) FROM shops WHERE name = ?",
			shop.name,
		).Scan(&count)

		if err != nil {
			log.Fatal(err)
		}

		if count > 0 {
			continue
		}

		_, err = DB.Exec(
			`INSERT INTO shops (
				name,
				location,
				phone,
				price_6kg,
				price_13kg,
				price_45kg
			)
			VALUES (?, ?, ?, ?, ?, ?)`,
			shop.name,
			shop.location,
			shop.phone,
			shop.price6,
			shop.price13,
			shop.price45,
		)

		if err != nil {
			log.Fatal(err)
		}
	}
}
