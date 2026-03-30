package dto

// AdvancePayload describes advance request body.
type AdvancePayload struct {
	RequestedAmount float64 `json:"requested_amount"`
	ApprovedAmount  float64 `json:"approved_amount,omitempty"`
	Currency        string  `json:"currency"`
	Comment         string  `json:"comment"`
}

// ExpenseReportPayload describes report update.
type ExpenseReportPayload struct {
	Currency string `json:"currency"`
}

// ExpenseReportStatusPayload describes status transitions.
type ExpenseReportStatusPayload struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

// ExpenseItemPayload describes expense item.
type ExpenseItemPayload struct {
	Category      string  `json:"category"`
	ExpenseDate   string  `json:"expense_date"`
	VendorName    string  `json:"vendor_name"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	TaxAmount     float64 `json:"tax_amount"`
	Description   string  `json:"description"`
	ReceiptFileID *string `json:"receipt_file_id"`
	Source        string  `json:"source"`
	Status        *string `json:"status,omitempty"`
}

// OCRDraftResponse describes OCR response.
type OCRDraftResponse struct {
	ExpenseDate string  `json:"expense_date"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Vendor      string  `json:"vendor"`
	Tax         float64 `json:"tax"`
	Confidence  float64 `json:"confidence"`
	Raw         any     `json:"raw"`
}
