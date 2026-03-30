package dto

// TripPayload is used for creating and updating trip requests.
type TripPayload struct {
	ProjectID             *string `json:"project_id"`
	DestinationCity       string  `json:"destination_city"`
	DestinationCountry    string  `json:"destination_country"`
	Purpose               string  `json:"purpose"`
	Comment               string  `json:"comment"`
	StartDate             string  `json:"start_date"`
	EndDate               string  `json:"end_date"`
	PlannedTransport      float64 `json:"planned_transport"`
	PlannedHotel          float64 `json:"planned_hotel"`
	PlannedDailyAllowance float64 `json:"planned_daily_allowance"`
	PlannedOther          float64 `json:"planned_other"`
	Currency              string  `json:"currency"`
	Status                *string `json:"status,omitempty"`
}

// StatusPayload contains only status to update.
type StatusPayload struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

// TripActionPayload handles submit/approve/reject actions.
type TripActionPayload struct {
	Action  string `json:"action"` // submit, approve, reject, request_changes, cancel
	Comment string `json:"comment"`
}
