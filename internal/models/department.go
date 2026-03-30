package models

import (
	"time"

	"github.com/google/uuid"
)

// Department represents a functional unit.
type Department struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
