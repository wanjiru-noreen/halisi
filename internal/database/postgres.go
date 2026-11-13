package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ConnectTimeout = 5 * time.Second

// Connect creates a bounded pool and verifies connectivity before returning it.
// Errors deliberately omit driver details, which may contain credentials.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database: DATABASE_URL is required")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("database: invalid DATABASE_URL or pool settings")
	}
	cfg.ConnConfig.ConnectTimeout = ConnectTimeout
	// URL pool settings may override pgx defaults; cap resource usage explicitly.
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MinIdleConns = 0
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("database: could not create connection pool")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("database: connection check failed; verify PostgreSQL availability and DATABASE_URL")
	}
	return pool, nil
}
