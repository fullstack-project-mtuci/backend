package models

import (
	"time"

	"github.com/google/uuid"
)

// Project groups work under budgets.
type Project struct {
	ID           uuid.UUID  `json:"id"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
