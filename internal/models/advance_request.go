package models

import (
	"time"

	"github.com/google/uuid"
)

// AdvanceStatus enumerates statuses for advance requests.
type AdvanceStatus string

const (
	AdvanceStatusDraft     AdvanceStatus = "draft"
	AdvanceStatusSubmitted AdvanceStatus = "submitted"
	AdvanceStatusApproved  AdvanceStatus = "approved"
	AdvanceStatusRejected  AdvanceStatus = "rejected"
	AdvanceStatusPaid      AdvanceStatus = "paid"
)

// AdvanceRequest is tied to a trip request.
type AdvanceRequest struct {
	ID              uuid.UUID     `json:"id"`
	TripRequestID   uuid.UUID     `json:"trip_request_id"`
	RequestedAmount float64       `json:"requested_amount"`
	ApprovedAmount  float64       `json:"approved_amount"`
	Currency        string        `json:"currency"`
	Status          AdvanceStatus `json:"status"`
	Comment         string        `json:"comment"`
	SubmittedAt     *time.Time    `json:"submitted_at,omitempty"`
	ProcessedAt     *time.Time    `json:"processed_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}
