package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
)

// ReferenceHandler exposes read-only reference data for authenticated users.
type ReferenceHandler struct {
	departments *repositories.DepartmentRepository
	references  *repositories.ReferenceRepository
}

// NewReferenceHandler constructs a ReferenceHandler.
func NewReferenceHandler(departments *repositories.DepartmentRepository, references *repositories.ReferenceRepository) *ReferenceHandler {
	return &ReferenceHandler{departments: departments, references: references}
}

// ListDepartments returns all departments.
func (h *ReferenceHandler) ListDepartments(c *fiber.Ctx) error {
	ctx := requestContext(c)
	deps, err := h.departments.ListDepartments(ctx)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": deps})
}

// ListProjects returns projects optionally filtered by department.
func (h *ReferenceHandler) ListProjects(c *fiber.Ctx) error {
	ctx := requestContext(c)
	user := middleware.GetUser(c)

	if user != nil && user.Role == models.RoleEmployee {
		if user.DepartmentID == nil {
			return fiber.NewError(fiber.StatusBadRequest, "department is not assigned to the current user")
		}
		projects, err := h.departments.ListProjectsForDepartmentWithBudget(ctx, *user.DepartmentID, time.Now())
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"items": projects})
	}

	var departmentID *uuid.UUID
	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid department_id")
		}
		departmentID = &id
	}

	projects, err := h.departments.ListProjects(ctx, departmentID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": projects})
}

// ListCategories returns available expense categories.
func (h *ReferenceHandler) ListCategories(c *fiber.Ctx) error {
	ctx := requestContext(c)
	cats, err := h.references.ListCategories(ctx)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": cats})
}
