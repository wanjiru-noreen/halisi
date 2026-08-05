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
		status TEXT NOT NULL
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
}

func seedShops() {

	var count int

	err := DB.QueryRow("SELECT COUNT(*) FROM shops").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	if count > 0 {
		return
	}

	shops := []struct {
		name     string
		location string
		phone    string
	}{
		{"Shell Gas", "Kisumu CBD", "0712345678"},
		{"Rubis Gas", "Kondele", "0723456789"},
		{"TotalEnergies", "Milimani", "0734567890"},
	}

	for _, shop := range shops {
		_, err := DB.Exec(
			`INSERT INTO shops(name, location, phone)
			 VALUES(?, ?, ?)`,
			shop.name,
			shop.location,
			shop.phone,
		)

		if err != nil {
			log.Fatal(err)
		}
	}
}
