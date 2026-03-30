package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

const expenseReportColumns = "id, trip_request_id, employee_id, advance_amount, total_expenses, balance_amount, currency, status, submitted_at, reviewed_at, closed_at, created_at, updated_at"

// ExpenseReportRepository persists expense reports.
type ExpenseReportRepository struct {
	pool *pgxpool.Pool
}

// NewExpenseReportRepository creates repository.
func NewExpenseReportRepository(pool *pgxpool.Pool) *ExpenseReportRepository {
	return &ExpenseReportRepository{pool: pool}
}

// Create inserts report.
func (r *ExpenseReportRepository) Create(ctx context.Context, report *models.ExpenseReport) error {
	const query = `
INSERT INTO expense_reports (trip_request_id, employee_id, advance_amount, total_expenses, balance_amount, currency, status)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING ` + expenseReportColumns
	return scanExpenseReport(r.pool.QueryRow(ctx, query,
		report.TripRequestID,
		report.EmployeeID,
		report.AdvanceAmount,
		report.TotalExpenses,
		report.BalanceAmount,
		report.Currency,
		report.Status,
	), report)
}

// Update stores modifications.
func (r *ExpenseReportRepository) Update(ctx context.Context, report *models.ExpenseReport) error {
	const query = `
UPDATE expense_reports
SET advance_amount = $2,
	total_expenses = $3,
	balance_amount = $4,
	currency = $5,
	status = $6,
	submitted_at = $7,
	reviewed_at = $8,
	closed_at = $9,
	updated_at = now()
WHERE id = $1
RETURNING ` + expenseReportColumns
	return scanExpenseReport(r.pool.QueryRow(ctx, query,
		report.ID,
		report.AdvanceAmount,
		report.TotalExpenses,
		report.BalanceAmount,
		report.Currency,
		report.Status,
		report.SubmittedAt,
		report.ReviewedAt,
		report.ClosedAt,
	), report)
}

// FindByTrip fetches report by trip.
func (r *ExpenseReportRepository) FindByTrip(ctx context.Context, tripID uuid.UUID) (*models.ExpenseReport, error) {
	var report models.ExpenseReport
	if err := scanExpenseReport(r.pool.QueryRow(ctx, `SELECT `+expenseReportColumns+` FROM expense_reports WHERE trip_request_id = $1`, tripID), &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// Get fetches by id.
func (r *ExpenseReportRepository) Get(ctx context.Context, id uuid.UUID) (*models.ExpenseReport, error) {
	var report models.ExpenseReport
	if err := scanExpenseReport(r.pool.QueryRow(ctx, `SELECT `+expenseReportColumns+` FROM expense_reports WHERE id = $1`, id), &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func scanExpenseReport(row pgx.Row, report *models.ExpenseReport) error {
	err := row.Scan(
		&report.ID,
		&report.TripRequestID,
		&report.EmployeeID,
		&report.AdvanceAmount,
		&report.TotalExpenses,
		&report.BalanceAmount,
		&report.Currency,
		&report.Status,
		&report.SubmittedAt,
		&report.ReviewedAt,
		&report.ClosedAt,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}
