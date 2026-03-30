package models

import (
	"time"

	"github.com/google/uuid"
)

// Role enumerates supported user roles.
type Role string

const (
	RoleEmployee   Role = "employee"
	RoleManager    Role = "manager"
	RoleAccountant Role = "accountant"
	RoleAdmin      Role = "admin"
)

// User is a person that can authenticate in the system.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Role         Role       `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	ManagerID    *uuid.UUID `json:"manager_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
