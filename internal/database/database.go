package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

//go:embed schema.sql migrations/0001_schema_normalization.up.sql
var databaseFiles embed.FS

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

	if err := createTables(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connected successfully")
}

func createTables() error {
	schema, err := databaseFiles.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read database schema: %w", err)
	}
	if _, err := DB.Exec(string(schema)); err != nil {
		return fmt.Errorf("apply database schema: %w", err)
	}

	migration, err := databaseFiles.ReadFile("migrations/0001_schema_normalization.up.sql")
	if err != nil {
		return fmt.Errorf("read schema normalization migration: %w", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("begin schema normalization migration: %w", err)
	}
	if _, err := tx.Exec(string(migration)); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("apply schema normalization migration: %w (rollback failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("apply schema normalization migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema normalization migration: %w", err)
	}
	return nil
}
