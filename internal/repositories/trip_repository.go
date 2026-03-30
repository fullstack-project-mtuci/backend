package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

const tripColumns = "id, employee_id, project_id, budget_id, destination_city, destination_country, purpose, comment, start_date, end_date, planned_transport, planned_hotel, planned_daily_allowance, planned_other, planned_total, currency, status, submitted_at, approved_at, rejected_at, created_at, updated_at, deleted_at"

// TripRepository handles persistence for trip requests.
type TripRepository struct {
	pool *pgxpool.Pool
}

// NewTripRepository builds TripRepository.
func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

// Create inserts a new trip request.
func (r *TripRepository) Create(ctx context.Context, trip *models.TripRequest) error {
	const query = `
INSERT INTO trip_requests (
	employee_id,
	project_id,
	budget_id,
	destination_city,
	destination_country,
	purpose,
	comment,
	start_date,
	end_date,
	planned_transport,
	planned_hotel,
	planned_daily_allowance,
	planned_other,
	planned_total,
	currency,
	status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
RETURNING ` + tripColumns

	return scanTrip(r.pool.QueryRow(ctx, query,
		trip.EmployeeID,
		trip.ProjectID,
		trip.BudgetID,
		trip.DestinationCity,
		trip.DestinationCountry,
		trip.Purpose,
		trip.Comment,
		trip.StartDate,
		trip.EndDate,
		trip.PlannedTransport,
		trip.PlannedHotel,
		trip.PlannedDailyAllowance,
		trip.PlannedOther,
		trip.PlannedTotal,
		trip.Currency,
		trip.Status,
	), trip)
}

// GetByID fetches a trip request by ID.
func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TripRequest, error) {
	const query = `SELECT ` + tripColumns + ` FROM trip_requests WHERE id = $1 AND deleted_at IS NULL`
	var trip models.TripRequest
	if err := scanTrip(r.pool.QueryRow(ctx, query, id), &trip); err != nil {
		return nil, err
	}
	return &trip, nil
}

// ListForEmployee returns trips for a specific employee.
func (r *TripRepository) ListForEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.TripRequest, error) {
	const query = `SELECT ` + tripColumns + ` FROM trip_requests WHERE employee_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTripRows(rows)
}

// ListAll returns all trip requests.
func (r *TripRepository) ListAll(ctx context.Context) ([]models.TripRequest, error) {
	const query = `SELECT ` + tripColumns + ` FROM trip_requests WHERE deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTripRows(rows)
}

// Update overwrites trip details.
func (r *TripRepository) Update(ctx context.Context, trip *models.TripRequest) error {
	const query = `
UPDATE trip_requests
SET project_id = $2,
	budget_id = $3,
	destination_city = $4,
	destination_country = $5,
	purpose = $6,
	comment = $7,
	start_date = $8,
	end_date = $9,
	planned_transport = $10,
	planned_hotel = $11,
	planned_daily_allowance = $12,
	planned_other = $13,
	planned_total = $14,
	currency = $15,
	status = $16,
	updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING ` + tripColumns

	return scanTrip(r.pool.QueryRow(ctx, query,
		trip.ID,
		trip.ProjectID,
		trip.BudgetID,
		trip.DestinationCity,
		trip.DestinationCountry,
		trip.Purpose,
		trip.Comment,
		trip.StartDate,
		trip.EndDate,
		trip.PlannedTransport,
		trip.PlannedHotel,
		trip.PlannedDailyAllowance,
		trip.PlannedOther,
		trip.PlannedTotal,
		trip.Currency,
		trip.Status,
	), trip)
}

// UpdateStatus changes the status of a trip request.
type TripStatusUpdate struct {
	SubmittedAt *time.Time
	ApprovedAt  *time.Time
	RejectedAt  *time.Time
	BudgetID    *uuid.UUID
}

func (r *TripRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TripStatus, update TripStatusUpdate) (*models.TripRequest, error) {
	const query = `
UPDATE trip_requests
SET status = $2,
	submitted_at = COALESCE($3, submitted_at),
	approved_at = COALESCE($4, approved_at),
	rejected_at = COALESCE($5, rejected_at),
	budget_id = COALESCE($6, budget_id),
	updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING ` + tripColumns

	var trip models.TripRequest
	if err := scanTrip(r.pool.QueryRow(ctx, query, id, status, update.SubmittedAt, update.ApprovedAt, update.RejectedAt, update.BudgetID), &trip); err != nil {
		return nil, err
	}
	return &trip, nil
}

// SoftDelete marks a trip as deleted.
func (r *TripRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE trip_requests
SET deleted_at = now(),
	updated_at = now()
WHERE id = $1 AND deleted_at IS NULL`

	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTrip(row pgx.Row, trip *models.TripRequest) error {
	err := row.Scan(
		&trip.ID,
		&trip.EmployeeID,
		&trip.ProjectID,
		&trip.BudgetID,
		&trip.DestinationCity,
		&trip.DestinationCountry,
		&trip.Purpose,
		&trip.Comment,
		&trip.StartDate,
		&trip.EndDate,
		&trip.PlannedTransport,
		&trip.PlannedHotel,
		&trip.PlannedDailyAllowance,
		&trip.PlannedOther,
		&trip.PlannedTotal,
		&trip.Currency,
		&trip.Status,
		&trip.SubmittedAt,
		&trip.ApprovedAt,
		&trip.RejectedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
		&trip.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func scanTripRows(rows pgx.Rows) ([]models.TripRequest, error) {
	var trips []models.TripRequest
	for rows.Next() {
		var trip models.TripRequest
		if err := scanTrip(rows, &trip); err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}
