package repositories

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// BudgetRepository manages budgets.
type BudgetRepository struct {
	pool *pgxpool.Pool
}

// NewBudgetRepository initializes repository.
func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{pool: pool}
}

// Create inserts a budget.
func (r *BudgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	const query = `
INSERT INTO budgets (scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id, scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency, created_at, updated_at`
	return scanBudget(r.pool.QueryRow(ctx, query,
		budget.ScopeType,
		budget.ScopeID,
		budget.PeriodStart,
		budget.PeriodEnd,
		budget.TotalLimit,
		budget.ReservedAmount,
		budget.SpentAmount,
		budget.Currency,
	), budget)
}

// Update adjusts totals.
func (r *BudgetRepository) Update(ctx context.Context, budget *models.Budget) error {
	const query = `
UPDATE budgets
SET total_limit = $2,
	reserved_amount = $3,
	spent_amount = $4,
	currency = $5,
	period_start = $6,
	period_end = $7,
	updated_at = now()
WHERE id = $1
RETURNING id, scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency, created_at, updated_at`
	return scanBudget(r.pool.QueryRow(ctx, query,
		budget.ID,
		budget.TotalLimit,
		budget.ReservedAmount,
		budget.SpentAmount,
		budget.Currency,
		budget.PeriodStart,
		budget.PeriodEnd,
	), budget)
}

// FindActiveBudget fetches a budget for given scope and date.
func (r *BudgetRepository) FindActiveBudget(ctx context.Context, scopeType models.BudgetScopeType, scopeID uuid.UUID, date time.Time) (*models.Budget, error) {
	const query = `
SELECT id, scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency, created_at, updated_at
FROM budgets
WHERE scope_type = $1 AND scope_id = $2 AND period_start <= $3 AND period_end >= $3
ORDER BY period_end DESC
LIMIT 1`
	var budget models.Budget
	if err := scanBudget(r.pool.QueryRow(ctx, query, scopeType, scopeID, date), &budget); err != nil {
		return nil, err
	}
	return &budget, nil
}

// List returns budgets filtered by scope.
func (r *BudgetRepository) List(ctx context.Context, scopeType *models.BudgetScopeType, scopeID *uuid.UUID) ([]models.Budget, error) {
	query := `SELECT id, scope_type, scope_id, period_start, period_end, total_limit, reserved_amount, spent_amount, currency, created_at, updated_at FROM budgets`
	args := []interface{}{}
	conditions := []string{}

	if scopeType != nil {
		conditions = append(conditions, `scope_type = $`+strconv.Itoa(len(args)+1))
		args = append(args, *scopeType)
	}
	if scopeID != nil {
		conditions = append(conditions, `scope_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *scopeID)
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		if err := scanBudget(rows, &budget); err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	return budgets, rows.Err()
}

// AdjustReserved changes reserved amount by delta.
func (r *BudgetRepository) AdjustReserved(ctx context.Context, id uuid.UUID, delta float64) error {
	const query = `
UPDATE budgets
SET reserved_amount = GREATEST(reserved_amount + $2, 0),
	updated_at = now()
WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id, delta)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Consume moves reserved funds to spent.
func (r *BudgetRepository) Consume(ctx context.Context, id uuid.UUID, amount float64) error {
	const query = `
UPDATE budgets
SET reserved_amount = GREATEST(reserved_amount - $2, 0),
	spent_amount = spent_amount + $2,
	updated_at = now()
WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id, amount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanBudget(row pgx.Row, budget *models.Budget) error {
	err := row.Scan(
		&budget.ID,
		&budget.ScopeType,
		&budget.ScopeID,
		&budget.PeriodStart,
		&budget.PeriodEnd,
		&budget.TotalLimit,
		&budget.ReservedAmount,
		&budget.SpentAmount,
		&budget.Currency,
		&budget.CreatedAt,
		&budget.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
