package models

import (
	"time"

	"github.com/google/uuid"
)

// TripStatus enumerates allowed statuses for a trip request.
type TripStatus string

const (
	TripStatusDraft            TripStatus = "draft"
	TripStatusSubmitted        TripStatus = "submitted"
	TripStatusManagerApproved  TripStatus = "manager_approved"
	TripStatusManagerRejected  TripStatus = "manager_rejected"
	TripStatusAccountantReview TripStatus = "accountant_review"
	TripStatusApproved         TripStatus = "approved"
	TripStatusCancelled        TripStatus = "cancelled"
)

// AllowedTripStatuses returns the list of supported trip statuses.
func AllowedTripStatuses() []TripStatus {
	return []TripStatus{
		TripStatusDraft,
		TripStatusSubmitted,
		TripStatusManagerApproved,
		TripStatusManagerRejected,
		TripStatusAccountantReview,
		TripStatusApproved,
		TripStatusCancelled,
	}
}

// TripRequest represents a business trip request made by an employee.
type TripRequest struct {
	ID                    uuid.UUID  `json:"id"`
	EmployeeID            uuid.UUID  `json:"employee_id"`
	ProjectID             *uuid.UUID `json:"project_id,omitempty"`
	BudgetID              *uuid.UUID `json:"budget_id,omitempty"`
	DestinationCity       string     `json:"destination_city"`
	DestinationCountry    string     `json:"destination_country"`
	Purpose               string     `json:"purpose"`
	StartDate             time.Time  `json:"start_date"`
	EndDate               time.Time  `json:"end_date"`
	Comment               string     `json:"comment"`
	PlannedTransport      float64    `json:"planned_transport"`
	PlannedHotel          float64    `json:"planned_hotel"`
	PlannedDailyAllowance float64    `json:"planned_daily_allowance"`
	PlannedOther          float64    `json:"planned_other"`
	PlannedTotal          float64    `json:"planned_total"`
	Currency              string     `json:"currency"`
	Status                TripStatus `json:"status"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt            *time.Time `json:"approved_at,omitempty"`
	RejectedAt            *time.Time `json:"rejected_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	DeletedAt             *time.Time `json:"-"`
}
