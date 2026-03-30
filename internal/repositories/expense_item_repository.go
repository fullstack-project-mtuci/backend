package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

const expenseItemColumns = "id, expense_report_id, category, expense_date, vendor_name, amount, currency, tax_amount, description, receipt_file_id, source, ocr_confidence, status, created_at, updated_at"

// ExpenseItemRepository handles expense items.
type ExpenseItemRepository struct {
	pool *pgxpool.Pool
}

// NewExpenseItemRepository creates repo.
func NewExpenseItemRepository(pool *pgxpool.Pool) *ExpenseItemRepository {
	return &ExpenseItemRepository{pool: pool}
}

// Create inserts new item.
func (r *ExpenseItemRepository) Create(ctx context.Context, item *models.ExpenseItem) error {
	const query = `
INSERT INTO expense_items (expense_report_id, category, expense_date, vendor_name, amount, currency, tax_amount, description, receipt_file_id, source, ocr_confidence, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING ` + expenseItemColumns
	return scanExpenseItem(r.pool.QueryRow(ctx, query,
		item.ExpenseReportID,
		item.Category,
		item.ExpenseDate,
		item.VendorName,
		item.Amount,
		item.Currency,
		item.TaxAmount,
		item.Description,
		item.ReceiptFileID,
		item.Source,
		item.OCRConfidence,
		item.Status,
	), item)
}

// Update modifies an item.
func (r *ExpenseItemRepository) Update(ctx context.Context, item *models.ExpenseItem) error {
	const query = `
UPDATE expense_items
SET category = $2,
	expense_date = $3,
	vendor_name = $4,
	amount = $5,
	currency = $6,
	tax_amount = $7,
	description = $8,
	receipt_file_id = $9,
	source = $10,
	ocr_confidence = $11,
	status = $12,
	updated_at = now()
WHERE id = $1
RETURNING ` + expenseItemColumns
	return scanExpenseItem(r.pool.QueryRow(ctx, query,
		item.ID,
		item.Category,
		item.ExpenseDate,
		item.VendorName,
		item.Amount,
		item.Currency,
		item.TaxAmount,
		item.Description,
		item.ReceiptFileID,
		item.Source,
		item.OCRConfidence,
		item.Status,
	), item)
}

// Get returns single expense item.
func (r *ExpenseItemRepository) Get(ctx context.Context, id uuid.UUID) (*models.ExpenseItem, error) {
	var item models.ExpenseItem
	if err := scanExpenseItem(r.pool.QueryRow(ctx, `SELECT `+expenseItemColumns+` FROM expense_items WHERE id = $1`, id), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// ListByReport returns items for report.
func (r *ExpenseItemRepository) ListByReport(ctx context.Context, reportID uuid.UUID) ([]models.ExpenseItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+expenseItemColumns+` FROM expense_items WHERE expense_report_id = $1 ORDER BY expense_date`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ExpenseItem
	for rows.Next() {
		var item models.ExpenseItem
		if err := scanExpenseItem(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Delete removes an item.
func (r *ExpenseItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM expense_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanExpenseItem(row pgx.Row, item *models.ExpenseItem) error {
	var expenseDate time.Time
	err := row.Scan(
		&item.ID,
		&item.ExpenseReportID,
		&item.Category,
		&expenseDate,
		&item.VendorName,
		&item.Amount,
		&item.Currency,
		&item.TaxAmount,
		&item.Description,
		&item.ReceiptFileID,
		&item.Source,
		&item.OCRConfidence,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	item.ExpenseDate = expenseDate
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}
