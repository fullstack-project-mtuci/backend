package models

import (
	"time"

	"github.com/google/uuid"
)

// ApprovalAction stores workflow decisions.
type ApprovalAction struct {
	ID         uuid.UUID `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Action     string    `json:"action"`
	ActorID    uuid.UUID `json:"actor_id"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuditLog tracks changes.
type AuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    uuid.UUID `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	BeforeJSON []byte    `json:"before_json"`
	AfterJSON  []byte    `json:"after_json"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}
