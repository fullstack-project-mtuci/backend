package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// ReferenceRepository manages expense categories and currencies.
type ReferenceRepository struct {
	pool *pgxpool.Pool
}

// NewReferenceRepository creates the repo.
func NewReferenceRepository(pool *pgxpool.Pool) *ReferenceRepository {
	return &ReferenceRepository{pool: pool}
}

// UpsertCategory creates or updates an expense category.
func (r *ReferenceRepository) UpsertCategory(ctx context.Context, cat *models.ExpenseCategory) error {
	const query = `
INSERT INTO expense_categories (id, name, code, is_active)
VALUES (COALESCE($1, gen_random_uuid()), $2, $3, $4)
ON CONFLICT (code)
DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now()
RETURNING id, name, code, is_active, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, cat.ID, cat.Name, cat.Code, cat.IsActive).Scan(
		&cat.ID, &cat.Name, &cat.Code, &cat.IsActive, &cat.CreatedAt, &cat.UpdatedAt,
	)
}

// ListCategories returns all categories.
func (r *ReferenceRepository) ListCategories(ctx context.Context) ([]models.ExpenseCategory, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, code, is_active, created_at, updated_at FROM expense_categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []models.ExpenseCategory
	for rows.Next() {
		var c models.ExpenseCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Code, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// DeleteCategory deletes category.
func (r *ReferenceRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertCurrency creates or updates currency reference.
func (r *ReferenceRepository) UpsertCurrency(ctx context.Context, currency *models.Currency) error {
	const query = `
INSERT INTO currencies (code, name, is_active)
VALUES ($1, $2, $3)
ON CONFLICT (code)
DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now()
RETURNING code, name, is_active, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, currency.Code, currency.Name, currency.IsActive).Scan(
		&currency.Code, &currency.Name, &currency.IsActive, &currency.CreatedAt, &currency.UpdatedAt,
	)
}

// ListCurrencies returns available currencies.
func (r *ReferenceRepository) ListCurrencies(ctx context.Context) ([]models.Currency, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name, is_active, created_at, updated_at FROM currencies ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var currencies []models.Currency
	for rows.Next() {
		var currency models.Currency
		if err := rows.Scan(&currency.Code, &currency.Name, &currency.IsActive, &currency.CreatedAt, &currency.UpdatedAt); err != nil {
			return nil, err
		}
		currencies = append(currencies, currency)
	}
	return currencies, rows.Err()
}
