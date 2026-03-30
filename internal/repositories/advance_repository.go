package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// AdvanceRepository persists advance requests.
type AdvanceRepository struct {
	pool *pgxpool.Pool
}

// NewAdvanceRepository builds repository.
func NewAdvanceRepository(pool *pgxpool.Pool) *AdvanceRepository {
	return &AdvanceRepository{pool: pool}
}

const advanceColumns = "id, trip_request_id, requested_amount, approved_amount, currency, status, comment, submitted_at, processed_at, created_at, updated_at"

// Create saves a new advance request.
func (r *AdvanceRepository) Create(ctx context.Context, adv *models.AdvanceRequest) error {
	const query = `
INSERT INTO advance_requests (trip_request_id, requested_amount, approved_amount, currency, status, comment)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING ` + advanceColumns
	return scanAdvance(r.pool.QueryRow(ctx, query,
		adv.TripRequestID,
		adv.RequestedAmount,
		adv.ApprovedAmount,
		adv.Currency,
		adv.Status,
		adv.Comment,
	), adv)
}

// Update saves modifications.
func (r *AdvanceRepository) Update(ctx context.Context, adv *models.AdvanceRequest) error {
	const query = `
UPDATE advance_requests
SET requested_amount = $2,
	approved_amount = $3,
	currency = $4,
	status = $5,
	comment = $6,
	submitted_at = $7,
	processed_at = $8,
	updated_at = now()
WHERE id = $1
RETURNING ` + advanceColumns
	return scanAdvance(r.pool.QueryRow(ctx, query,
		adv.ID,
		adv.RequestedAmount,
		adv.ApprovedAmount,
		adv.Currency,
		adv.Status,
		adv.Comment,
		adv.SubmittedAt,
		adv.ProcessedAt,
	), adv)
}

// FindByTrip returns advance by trip id.
func (r *AdvanceRepository) FindByTrip(ctx context.Context, tripID uuid.UUID) (*models.AdvanceRequest, error) {
	var adv models.AdvanceRequest
	if err := scanAdvance(r.pool.QueryRow(ctx, `SELECT `+advanceColumns+` FROM advance_requests WHERE trip_request_id = $1`, tripID), &adv); err != nil {
		return nil, err
	}
	return &adv, nil
}

// Get returns advance by id.
func (r *AdvanceRepository) Get(ctx context.Context, id uuid.UUID) (*models.AdvanceRequest, error) {
	var adv models.AdvanceRequest
	if err := scanAdvance(r.pool.QueryRow(ctx, `SELECT `+advanceColumns+` FROM advance_requests WHERE id = $1`, id), &adv); err != nil {
		return nil, err
	}
	return &adv, nil
}

func scanAdvance(row pgx.Row, adv *models.AdvanceRequest) error {
	err := row.Scan(
		&adv.ID,
		&adv.TripRequestID,
		&adv.RequestedAmount,
		&adv.ApprovedAmount,
		&adv.Currency,
		&adv.Status,
		&adv.Comment,
		&adv.SubmittedAt,
		&adv.ProcessedAt,
		&adv.CreatedAt,
		&adv.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	return err
}
