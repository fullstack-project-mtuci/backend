package models

import (
	"time"

	"github.com/google/uuid"
)

// ReceiptFile stores metadata for uploaded receipt.
type ReceiptFile struct {
	ID               uuid.UUID `json:"id"`
	UploadedBy       uuid.UUID `json:"uploaded_by"`
	StoragePath      string    `json:"storage_path"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	Checksum         string    `json:"checksum"`
	CreatedAt        time.Time `json:"created_at"`
}

// ReceiptRecognition stores OCR results.
type ReceiptRecognition struct {
	ID                uuid.UUID  `json:"id"`
	ReceiptFileID     uuid.UUID  `json:"receipt_file_id"`
	RawResponseJSON   []byte     `json:"raw_response_json"`
	ExtractedDate     *time.Time `json:"extracted_date,omitempty"`
	ExtractedAmount   *float64   `json:"extracted_amount,omitempty"`
	ExtractedCurrency string     `json:"extracted_currency,omitempty"`
	ExtractedVendor   string     `json:"extracted_vendor,omitempty"`
	ExtractedTax      *float64   `json:"extracted_tax,omitempty"`
	ConfidenceScore   *float64   `json:"confidence_score,omitempty"`
	Status            string     `json:"status"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
