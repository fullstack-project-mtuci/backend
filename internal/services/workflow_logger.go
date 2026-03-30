package services

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"backend/internal/models"
	"backend/internal/repositories"
)

// WorkflowLogger persists approval and audit actions.
type WorkflowLogger struct {
	approvals *repositories.ApprovalRepository
	audits    *repositories.AuditRepository
}

// NewWorkflowLogger constructs logger service.
func NewWorkflowLogger(approvals *repositories.ApprovalRepository, audits *repositories.AuditRepository) *WorkflowLogger {
	return &WorkflowLogger{approvals: approvals, audits: audits}
}

// Approval records workflow decision.
func (l *WorkflowLogger) Approval(ctx context.Context, entityType string, entityID uuid.UUID, action string, actorID uuid.UUID, comment string) error {
	return l.approvals.AddAction(ctx, &models.ApprovalAction{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		Comment:    comment,
	})
}

// Audit records state change.
func (l *WorkflowLogger) Audit(ctx context.Context, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, before interface{}, after interface{}, ip, agent string) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	return l.audits.AddLog(ctx, &models.AuditLog{
		ActorID:    actorID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
		IPAddress:  ip,
		UserAgent:  agent,
	})
}
