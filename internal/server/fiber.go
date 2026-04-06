package server

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"

	"backend/internal/config"
)

// NewFiberApp configures Fiber instance with common middleware and error handler.
func NewFiberApp(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if fe, ok := err.(*fiber.Error); ok {
				return c.Status(fe.Code).JSON(fiber.Map{
					"error":   fe.Message,
					"status":  fe.Code,
					"success": false,
				})
			}

			log.Printf("unhandled error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal server error",
				"status":  fiber.StatusInternalServerError,
				"success": false,
			})
		},
	})

	app.Use(fiberRecover.New())
	app.Use(fiberLogger.New())
	allowedOrigins := strings.Join(cfg.CORS.AllowedOrigins, ",")
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: cfg.CORS.AllowCredentials,
	}))

	return app
}
