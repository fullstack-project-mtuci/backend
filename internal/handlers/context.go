package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// requestContext returns the request-scoped context or background.
func requestContext(c *fiber.Ctx) context.Context {
	if ctx := c.UserContext(); ctx != nil {
		return ctx
	}
	return context.Background()
}
