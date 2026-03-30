package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// AuditRepository persists audit logs.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates repository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// AddLog stores entry.
func (r *AuditRepository) AddLog(ctx context.Context, logEntry *models.AuditLog) error {
	const query = `
INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id, actor_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent, created_at`
	return r.pool.QueryRow(ctx, query,
		logEntry.ActorID,
		logEntry.Action,
		logEntry.EntityType,
		logEntry.EntityID,
		logEntry.BeforeJSON,
		logEntry.AfterJSON,
		logEntry.IPAddress,
		logEntry.UserAgent,
	).Scan(
		&logEntry.ID,
		&logEntry.ActorID,
		&logEntry.Action,
		&logEntry.EntityType,
		&logEntry.EntityID,
		&logEntry.BeforeJSON,
		&logEntry.AfterJSON,
		&logEntry.IPAddress,
		&logEntry.UserAgent,
		&logEntry.CreatedAt,
	)
}

// ListLogs returns logs for entity.
func (r *AuditRepository) ListLogs(ctx context.Context, entityType string, entityID uuid.UUID) ([]models.AuditLog, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, actor_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent, created_at FROM audit_logs WHERE entity_type = $1 AND entity_id = $2 ORDER BY created_at DESC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(
			&entry.ID,
			&entry.ActorID,
			&entry.Action,
			&entry.EntityType,
			&entry.EntityID,
			&entry.BeforeJSON,
			&entry.AfterJSON,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}
