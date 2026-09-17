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

	if !columns["latitude"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN latitude REAL DEFAULT 0
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["longitude"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN longitude REAL DEFAULT 0
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["business_registration_number"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN business_registration_number TEXT DEFAULT ''
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["kra_pin"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN kra_pin TEXT DEFAULT ''
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["license_number"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN license_number TEXT DEFAULT ''
		`)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !columns["verification_status"] {
		_, err := DB.Exec(`
			ALTER TABLE shops
			ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'pending'
		`)

		if err != nil {
			log.Fatal(err)
		}
	}
}
