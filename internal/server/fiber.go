package server

import (
	"log"

	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"
)

// NewFiberApp configures Fiber instance with common middleware and error handler.
func NewFiberApp() *fiber.App {
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

	return app
}
