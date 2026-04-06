package repositories

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// ReceiptRepository manages receipt files and OCR entries.
type ReceiptRepository struct {
	pool *pgxpool.Pool
}

// NewReceiptRepository builds repository.
func NewReceiptRepository(pool *pgxpool.Pool) *ReceiptRepository {
	return &ReceiptRepository{pool: pool}
}

const receiptColumns = "id, uploaded_by, storage_path, original_filename, mime_type, file_size, checksum, created_at"

// CreateFile stores metadata about uploaded file.
func (r *ReceiptRepository) CreateFile(ctx context.Context, file *models.ReceiptFile) error {
	const query = `
INSERT INTO receipt_files (uploaded_by, storage_path, original_filename, mime_type, file_size, checksum)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING ` + receiptColumns
	return r.pool.QueryRow(ctx, query,
		file.UploadedBy,
		file.StoragePath,
		file.OriginalFilename,
		file.MimeType,
		file.FileSize,
		file.Checksum,
	).Scan(
		&file.ID,
		&file.UploadedBy,
		&file.StoragePath,
		&file.OriginalFilename,
		&file.MimeType,
		&file.FileSize,
		&file.Checksum,
		&file.CreatedAt,
	)
}

// GetFile fetches metadata.
func (r *ReceiptRepository) GetFile(ctx context.Context, id uuid.UUID) (*models.ReceiptFile, error) {
	var file models.ReceiptFile
	err := r.pool.QueryRow(ctx, `SELECT `+receiptColumns+` FROM receipt_files WHERE id = $1`, id).Scan(
		&file.ID,
		&file.UploadedBy,
		&file.StoragePath,
		&file.OriginalFilename,
		&file.MimeType,
		&file.FileSize,
		&file.Checksum,
		&file.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return &file, err
}

// CreateRecognition stores OCR results.
func (r *ReceiptRepository) CreateRecognition(ctx context.Context, rec *models.ReceiptRecognition) error {
	const query = `
INSERT INTO receipt_recognitions (receipt_file_id, raw_response_json, extracted_date, extracted_amount, extracted_currency, extracted_vendor, extracted_tax, confidence_score, status, processed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, receipt_file_id, raw_response_json, extracted_date, extracted_amount, extracted_currency, extracted_vendor, extracted_tax, confidence_score, status, processed_at, created_at`
	return r.pool.QueryRow(ctx, query,
		rec.ReceiptFileID,
		rec.RawResponseJSON,
		rec.ExtractedDate,
		rec.ExtractedAmount,
		rec.ExtractedCurrency,
		rec.ExtractedVendor,
		rec.ExtractedTax,
		rec.ConfidenceScore,
		rec.Status,
		rec.ProcessedAt,
	).Scan(
		&rec.ID,
		&rec.ReceiptFileID,
		&rec.RawResponseJSON,
		&rec.ExtractedDate,
		&rec.ExtractedAmount,
		&rec.ExtractedCurrency,
		&rec.ExtractedVendor,
		&rec.ExtractedTax,
		&rec.ConfidenceScore,
		&rec.Status,
		&rec.ProcessedAt,
		&rec.CreatedAt,
	)
}

// LatestRecognition fetches last OCR result for file.
func (r *ReceiptRepository) LatestRecognition(ctx context.Context, fileID uuid.UUID) (*models.ReceiptRecognition, error) {
	const query = `
SELECT id, receipt_file_id, raw_response_json, extracted_date, extracted_amount, extracted_currency, extracted_vendor, extracted_tax, confidence_score, status, processed_at, created_at
FROM receipt_recognitions
WHERE receipt_file_id = $1
ORDER BY created_at DESC
LIMIT 1`
	var rec models.ReceiptRecognition
	err := r.pool.QueryRow(ctx, query, fileID).Scan(
		&rec.ID,
		&rec.ReceiptFileID,
		&rec.RawResponseJSON,
		&rec.ExtractedDate,
		&rec.ExtractedAmount,
		&rec.ExtractedCurrency,
		&rec.ExtractedVendor,
		&rec.ExtractedTax,
		&rec.ConfidenceScore,
		&rec.Status,
		&rec.ProcessedAt,
		&rec.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return &rec, err
}

// ListFilesByUser returns files uploaded by user.
func (r *ReceiptRepository) ListFilesByUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReceiptFile, error) {
	query := `SELECT ` + receiptColumns + ` FROM receipt_files WHERE uploaded_by = $1 ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ` + strconv.Itoa(limit)
	}
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.ReceiptFile
	for rows.Next() {
		var file models.ReceiptFile
		if err := rows.Scan(
			&file.ID,
			&file.UploadedBy,
			&file.StoragePath,
			&file.OriginalFilename,
			&file.MimeType,
			&file.FileSize,
			&file.Checksum,
			&file.CreatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}
