package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// ApprovalRepository stores approval actions and audit logs.
type ApprovalRepository struct {
	pool *pgxpool.Pool
}

// NewApprovalRepository creates repository.
func NewApprovalRepository(pool *pgxpool.Pool) *ApprovalRepository {
	return &ApprovalRepository{pool: pool}
}

// AddAction saves approval action.
func (r *ApprovalRepository) AddAction(ctx context.Context, action *models.ApprovalAction) error {
	const query = `
INSERT INTO approval_actions (entity_type, entity_id, action, actor_id, comment)
VALUES ($1,$2,$3,$4,$5)
RETURNING id, entity_type, entity_id, action, actor_id, comment, created_at`
	return r.pool.QueryRow(ctx, query,
		action.EntityType,
		action.EntityID,
		action.Action,
		action.ActorID,
		action.Comment,
	).Scan(
		&action.ID,
		&action.EntityType,
		&action.EntityID,
		&action.Action,
		&action.ActorID,
		&action.Comment,
		&action.CreatedAt,
	)
}

// ListActions returns actions for entity.
func (r *ApprovalRepository) ListActions(ctx context.Context, entityType string, entityID uuid.UUID) ([]models.ApprovalAction, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, entity_type, entity_id, action, actor_id, comment, created_at FROM approval_actions WHERE entity_type=$1 AND entity_id=$2 ORDER BY created_at DESC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []models.ApprovalAction
	for rows.Next() {
		var action models.ApprovalAction
		if err := rows.Scan(
			&action.ID,
			&action.EntityType,
			&action.EntityID,
			&action.Action,
			&action.ActorID,
			&action.Comment,
			&action.CreatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}
