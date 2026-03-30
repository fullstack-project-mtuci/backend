package models

import (
	"time"

	"github.com/google/uuid"
)

// ExpenseReportStatus enumerates statuses.
type ExpenseReportStatus string

const (
	ExpenseReportDraft            ExpenseReportStatus = "draft"
	ExpenseReportSubmitted        ExpenseReportStatus = "submitted"
	ExpenseReportManagerReview    ExpenseReportStatus = "manager_review"
	ExpenseReportAccountantReview ExpenseReportStatus = "accountant_review"
	ExpenseReportNeedsRevision    ExpenseReportStatus = "needs_revision"
	ExpenseReportApproved         ExpenseReportStatus = "approved"
	ExpenseReportRejected         ExpenseReportStatus = "rejected"
	ExpenseReportClosed           ExpenseReportStatus = "closed"
)

// ExpenseReport collects expenses for a trip.
type ExpenseReport struct {
	ID            uuid.UUID           `json:"id"`
	TripRequestID uuid.UUID           `json:"trip_request_id"`
	EmployeeID    uuid.UUID           `json:"employee_id"`
	AdvanceAmount float64             `json:"advance_amount"`
	TotalExpenses float64             `json:"total_expenses"`
	BalanceAmount float64             `json:"balance_amount"`
	Currency      string              `json:"currency"`
	Status        ExpenseReportStatus `json:"status"`
	SubmittedAt   *time.Time          `json:"submitted_at,omitempty"`
	ReviewedAt    *time.Time          `json:"reviewed_at,omitempty"`
	ClosedAt      *time.Time          `json:"closed_at,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}
