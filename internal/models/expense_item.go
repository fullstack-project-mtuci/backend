package models

import (
	"time"

	"github.com/google/uuid"
)

// ExpenseItemStatus enumerates statuses for expense items.
type ExpenseItemStatus string

const (
	ExpenseItemDraft         ExpenseItemStatus = "draft"
	ExpenseItemPendingReview ExpenseItemStatus = "pending_review"
	ExpenseItemAccepted      ExpenseItemStatus = "accepted"
	ExpenseItemRejected      ExpenseItemStatus = "rejected"
)

// ExpenseSource indicates origin of data.
type ExpenseSource string

const (
	ExpenseSourceManual   ExpenseSource = "manual"
	ExpenseSourceOCRDraft ExpenseSource = "ocr_draft"
)

// ExpenseItem describes a single expense entry.
type ExpenseItem struct {
	ID              uuid.UUID         `json:"id"`
	ExpenseReportID uuid.UUID         `json:"expense_report_id"`
	Category        string            `json:"category"`
	ExpenseDate     time.Time         `json:"expense_date"`
	VendorName      string            `json:"vendor_name"`
	Amount          float64           `json:"amount"`
	Currency        string            `json:"currency"`
	TaxAmount       float64           `json:"tax_amount"`
	Description     string            `json:"description"`
	ReceiptFileID   *uuid.UUID        `json:"receipt_file_id,omitempty"`
	Source          ExpenseSource     `json:"source"`
	OCRConfidence   *float64          `json:"ocr_confidence,omitempty"`
	Status          ExpenseItemStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
