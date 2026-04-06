package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/models"
)

// DepartmentRepository handles departments and projects.
type DepartmentRepository struct {
	pool *pgxpool.Pool
}

// NewDepartmentRepository creates the repository.
func NewDepartmentRepository(pool *pgxpool.Pool) *DepartmentRepository {
	return &DepartmentRepository{pool: pool}
}

// CreateDepartment inserts a department.
func (r *DepartmentRepository) CreateDepartment(ctx context.Context, d *models.Department) error {
	const query = `
INSERT INTO departments (name, code)
VALUES ($1, $2)
RETURNING id, name, code, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, d.Name, d.Code).Scan(
		&d.ID, &d.Name, &d.Code, &d.CreatedAt, &d.UpdatedAt,
	)
}

// UpdateDepartment updates department fields.
func (r *DepartmentRepository) UpdateDepartment(ctx context.Context, d *models.Department) error {
	const query = `
UPDATE departments
SET name = $2,
	code = $3,
	updated_at = now()
WHERE id = $1
RETURNING id, name, code, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, d.ID, d.Name, d.Code).Scan(
		&d.ID, &d.Name, &d.Code, &d.CreatedAt, &d.UpdatedAt,
	)
}

// ListDepartments returns all departments.
func (r *DepartmentRepository) ListDepartments(ctx context.Context) ([]models.Department, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, code, created_at, updated_at FROM departments ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

// DeleteDepartment removes department if no references.
func (r *DepartmentRepository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateProject inserts a project.
func (r *DepartmentRepository) CreateProject(ctx context.Context, p *models.Project) error {
	const query = `
INSERT INTO projects (name, code, department_id, is_active)
VALUES ($1,$2,$3,$4)
RETURNING id, department_id, name, code, is_active, created_at, updated_at`
	return scanProject(r.pool.QueryRow(ctx, query, p.Name, p.Code, p.DepartmentID, p.IsActive), p)
}

// UpdateProject updates project fields.
func (r *DepartmentRepository) UpdateProject(ctx context.Context, p *models.Project) error {
	const query = `
UPDATE projects
SET name = $2,
	code = $3,
	department_id = $4,
	is_active = $5,
	updated_at = now()
WHERE id = $1
RETURNING id, department_id, name, code, is_active, created_at, updated_at`
	return scanProject(r.pool.QueryRow(ctx, query, p.ID, p.Name, p.Code, p.DepartmentID, p.IsActive), p)
}

// ListProjects returns projects optionally filtered by department.
func (r *DepartmentRepository) ListProjects(ctx context.Context, departmentID *uuid.UUID) ([]models.Project, error) {
	query := `SELECT id, department_id, name, code, is_active, created_at, updated_at FROM projects`
	args := []interface{}{}
	if departmentID != nil {
		query += ` WHERE department_id = $1`
		args = append(args, *departmentID)
	}
	query += ` ORDER BY name`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := scanProject(rows, &p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// DeleteProject removes a project.
func (r *DepartmentRepository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanProject(row pgx.Row, p *models.Project) error {
	return row.Scan(&p.ID, &p.DepartmentID, &p.Name, &p.Code, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
}

// ListProjectsForDepartmentWithBudget returns active projects for department with available budgets.
func (r *DepartmentRepository) ListProjectsForDepartmentWithBudget(ctx context.Context, departmentID uuid.UUID, at time.Time) ([]models.Project, error) {
	const query = `
SELECT DISTINCT p.id, p.department_id, p.name, p.code, p.is_active, p.created_at, p.updated_at
FROM projects p
LEFT JOIN budgets pb
	ON pb.scope_type = 'project'
	AND pb.scope_id = p.id
	AND pb.period_start <= $2
	AND pb.period_end >= $2
LEFT JOIN budgets db
	ON db.scope_type = 'department'
	AND db.scope_id = p.department_id
	AND db.period_start <= $2
	AND db.period_end >= $2
WHERE p.department_id = $1
	AND p.is_active = TRUE
	AND (pb.id IS NOT NULL OR db.id IS NOT NULL)
ORDER BY p.name`

	rows, err := r.pool.Query(ctx, query, departmentID, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := scanProject(rows, &p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}
