package dto

// DepartmentPayload describes department create/update request.
type DepartmentPayload struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// ProjectPayload describes project data.
type ProjectPayload struct {
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	DepartmentID *string `json:"department_id"`
	IsActive     bool    `json:"is_active"`
}

// UserPayload describes user admin request.
type UserPayload struct {
	Email        string  `json:"email"`
	FullName     string  `json:"full_name"`
	Role         string  `json:"role"`
	DepartmentID *string `json:"department_id"`
	ManagerID    *string `json:"manager_id"`
	IsActive     *bool   `json:"is_active"`
	Password     *string `json:"password,omitempty"`
}

// BudgetPayload describes budget info.
type BudgetPayload struct {
	ScopeType   string  `json:"scope_type"`
	ScopeID     string  `json:"scope_id"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	TotalLimit  float64 `json:"total_limit"`
	Currency    string  `json:"currency"`
}

// ExpenseCategoryPayload describes category info.
type ExpenseCategoryPayload struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	IsActive bool   `json:"is_active"`
}
