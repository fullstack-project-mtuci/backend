// @title Travel Portal API
// @version 1.0
// @description REST API for the Business Trip and Expense Portal
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log"

	"backend/internal/app"
)

func main() {
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}
