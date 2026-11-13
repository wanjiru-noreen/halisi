package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationFiles(t *testing.T) {
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	version, err := source.First()
	if err != nil || version != 1 {
		t.Fatal("expected initial migration version 1")
	}
	up, _, err := source.ReadUp(version)
	if err != nil {
		t.Fatal(err)
	}
	up.Close()
	down, _, err := source.ReadDown(version)
	if err != nil {
		t.Fatal(err)
	}
	down.Close()
}

func TestMigratorRejectsInvalidURL(t *testing.T) {
	for _, value := range []string{"", "https://localhost/db", "postgres:///db", "postgres://user:secret%zz@localhost/db"} {
		m, err := NewMigrator(value)
		if err == nil || m != nil {
			t.Fatal("expected URL validation error")
		}
		if err.Error() != "migrations: DATABASE_URL must be a PostgreSQL URL with a host" {
			t.Fatal("unexpected validation error or leaked credentials")
		}
	}
}

func TestMigrationsPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_MIGRATION_DATABASE_URL to enable live migration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("cannot create integration test pool")
	}
	defer admin.Close()
	schema := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal("cannot create isolated test schema")
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("could not remove integration test schema")
		}
	}()
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid integration database URL")
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	m, err := NewMigrator(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatal("initial migration failed")
	}
	version, dirty, err := m.Version()
	if err != nil || version != 1 || dirty {
		t.Fatal("expected clean version 1")
	}
	for _, table := range []string{"users", "vendors", "products", "orders", "order_items", "payments", "deliveries", "schema_migrations"} {
		var exists bool
		if err := admin.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || !exists {
			t.Fatalf("expected table %s", table)
		}
	}
	// Persist a row to prove repeat-up does not recreate the schema or lose data.
	if _, err := admin.Exec(ctx, "INSERT INTO "+schema+".users (name, email, password_hash, role) VALUES ('test', 'test@example.invalid', 'not-a-real-hash', 'customer')"); err != nil {
		t.Fatal("cannot insert migration test row")
	}
	if err := m.Up(); !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal("repeat up should report no change")
	}
	var count int
	if err := admin.QueryRow(ctx, "SELECT count(*) FROM "+schema+".users").Scan(&count); err != nil || count != 1 {
		t.Fatal("repeat up changed existing data")
	}
	if err := m.Steps(-1); err != nil {
		t.Fatal("rollback failed")
	}
	if _, _, err := m.Version(); !errors.Is(err, migrate.ErrNilVersion) {
		t.Fatal("expected no applied version after rollback")
	}
	for _, table := range []string{"users", "vendors", "products", "orders", "order_items", "payments", "deliveries"} {
		var exists bool
		if err := admin.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || exists {
			t.Fatalf("rollback did not remove %s", table)
		}
	}
	if err := m.Up(); err != nil {
		t.Fatal("reapply after rollback failed")
	}
}
