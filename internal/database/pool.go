package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/config"
)

// NewPool creates a PostgreSQL connection pool using pgx.
func NewPool(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
