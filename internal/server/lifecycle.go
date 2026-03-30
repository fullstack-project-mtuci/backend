package server

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"backend/internal/config"
)

// RegisterFiberLifecycle wires Fiber server start/stop to Fx lifecycle.
func RegisterFiberLifecycle(lc fx.Lifecycle, app *fiber.App, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("Starting server on :%s", cfg.Port)
				if err := app.Listen(":" + cfg.Port); err != nil && err != fiber.ErrServiceUnavailable {
					log.Printf("fiber server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return app.ShutdownWithContext(shutdownCtx)
		},
	})
}

// RegisterDatabaseLifecycle closes DB pool when application stops.
func RegisterDatabaseLifecycle(lc fx.Lifecycle, pool *pgxpool.Pool) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})
}
