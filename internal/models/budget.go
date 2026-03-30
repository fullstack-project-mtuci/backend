package models

import (
	"time"

	"github.com/google/uuid"
)

// BudgetScopeType defines budget scoping.
type BudgetScopeType string

const (
	BudgetScopeDepartment BudgetScopeType = "department"
	BudgetScopeProject    BudgetScopeType = "project"
)

// Budget stores financial allocation.
type Budget struct {
	ID             uuid.UUID       `json:"id"`
	ScopeType      BudgetScopeType `json:"scope_type"`
	ScopeID        uuid.UUID       `json:"scope_id"`
	PeriodStart    time.Time       `json:"period_start"`
	PeriodEnd      time.Time       `json:"period_end"`
	TotalLimit     float64         `json:"total_limit"`
	ReservedAmount float64         `json:"reserved_amount"`
	SpentAmount    float64         `json:"spent_amount"`
	Currency       string          `json:"currency"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
