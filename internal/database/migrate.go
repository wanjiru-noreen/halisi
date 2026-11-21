package database

import (
	"embed"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// NewMigrator uses embedded, version-controlled SQL and the configured database.
// The caller must Close the migrator when finished.
func NewMigrator(databaseURL string) (*migrate.Migrate, error) {
	u, err := url.Parse(databaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" {
		return nil, errors.New("migrations: DATABASE_URL must be a PostgreSQL URL with a host")
	}
	if strings.TrimSpace(os.Getenv("APP_ENV")) == "production" {
		pgCfg, err := pgconn.ParseConfig(databaseURL)
		if err != nil {
			return nil, errors.New("migrations: DATABASE_URL must be a valid PostgreSQL URL")
		}
		if pgCfg.TLSConfig == nil {
			return nil, errors.New("migrations: TLS is required in production (sslmode=require)")
		}
		for _, fb := range pgCfg.Fallbacks {
			if fb.TLSConfig == nil {
				return nil, errors.New("migrations: TLS is required in production (sslmode=require)")
			}
		}
	}
	u.Scheme = "pgx5"
	query := u.Query()
	if strings.TrimSpace(os.Getenv("APP_ENV")) == "production" {
		sslmode := strings.ToLower(strings.TrimSpace(query.Get("sslmode")))
		if sslmode != "require" && sslmode != "verify-ca" && sslmode != "verify-full" {
			query.Set("sslmode", "require")
		}
	}
	if !query.Has("connect_timeout") {
		query.Set("connect_timeout", "5")
	}
	u.RawQuery = query.Encode()
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, errors.New("migrations: could not open embedded SQL")
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, u.String())
	if err != nil {
		_ = source.Close()
		return nil, errors.New("migrations: cannot initialize; check PostgreSQL availability, DATABASE_URL and schema permissions")
	}
	return m, nil
}
