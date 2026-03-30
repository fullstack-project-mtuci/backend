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
