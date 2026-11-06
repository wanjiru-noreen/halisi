package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

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

	if err := runMigrations(databaseURL); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	log.Println("Database connected successfully")
}

func runMigrations(databaseURL string) error {
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	migrationDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return errors.Join(
			fmt.Errorf("open migration database: %w", err),
			source.Close(),
		)
	}

	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		return errors.Join(
			fmt.Errorf("create migration database driver: %w", err),
			source.Close(),
			migrationDB.Close(),
		)
	}

	runner, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return errors.Join(
			fmt.Errorf("create migration runner: %w", err),
			source.Close(),
			driver.Close(),
		)
	}

	err = runner.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("apply migrations: %w", err)
	}

	sourceErr, databaseErr := runner.Close()
	return errors.Join(err, sourceErr, databaseErr)
}
