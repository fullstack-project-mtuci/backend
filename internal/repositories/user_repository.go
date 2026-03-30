package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

const userColumns = "id, email, password_hash, full_name, role, department_id, manager_id, is_active, created_at, updated_at"

// UserRepository handles persistence for users.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository initializes UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and populates the struct with DB values.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	const query = `
INSERT INTO users (email, password_hash, full_name, role, department_id, manager_id, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + userColumns

	return scanUser(r.pool.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.DepartmentID,
		user.ManagerID,
		user.IsActive,
	), user)
}

// FindByEmail finds user by email or returns ErrNotFound.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	var user models.User
	if err := scanUser(r.pool.QueryRow(ctx, query, email), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds user by id or returns ErrNotFound.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	var user models.User
	if err := scanUser(r.pool.QueryRow(ctx, query, id), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// List returns users filtered by role/department/active flag.
func (r *UserRepository) List(ctx context.Context, params UserListParams) ([]models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if params.Role != "" {
		query += fmt.Sprintf(" AND role = $%d", argPos)
		args = append(args, params.Role)
		argPos++
	}

	if params.DepartmentID != nil {
		query += fmt.Sprintf(" AND department_id = $%d", argPos)
		args = append(args, *params.DepartmentID)
		argPos++
	}

	if params.IncludeInactive {
		// no filter
	} else {
		query += fmt.Sprintf(" AND is_active = true")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// UpdateRole updates basic user fields.
func (r *UserRepository) UpdateRole(ctx context.Context, user *models.User) error {
	const query = `
UPDATE users
SET full_name = $2,
	role = $3,
	department_id = $4,
	manager_id = $5,
	is_active = $6,
	updated_at = now()
WHERE id = $1
RETURNING ` + userColumns

	return scanUser(r.pool.QueryRow(ctx, query,
		user.ID,
		user.FullName,
		user.Role,
		user.DepartmentID,
		user.ManagerID,
		user.IsActive,
	), user)
}

// SetPassword updates password hash.
func (r *UserRepository) SetPassword(ctx context.Context, userID uuid.UUID, hash string) error {
	const query = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, userID, hash)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UserListParams contains filters for list query.
type UserListParams struct {
	Role            models.Role
	DepartmentID    *uuid.UUID
	IncludeInactive bool
}

func scanUser(row pgx.Row, user *models.User) error {
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.DepartmentID,
		&user.ManagerID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
