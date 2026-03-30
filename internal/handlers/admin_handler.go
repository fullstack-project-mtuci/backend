package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"backend/internal/dto"
	"backend/internal/models"
	"backend/internal/repositories"
)

// AdminHandler manages administrative endpoints.
type AdminHandler struct {
	users       *repositories.UserRepository
	departments *repositories.DepartmentRepository
	references  *repositories.ReferenceRepository
	budgets     *repositories.BudgetRepository
}

// NewAdminHandler constructs AdminHandler.
func NewAdminHandler(
	users *repositories.UserRepository,
	departments *repositories.DepartmentRepository,
	references *repositories.ReferenceRepository,
	budgets *repositories.BudgetRepository,
) *AdminHandler {
	return &AdminHandler{
		users:       users,
		departments: departments,
		references:  references,
		budgets:     budgets,
	}
}

// ListUsers returns users with optional filters.
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	var params repositories.UserListParams
	role := strings.TrimSpace(c.Query("role"))
	if role != "" {
		params.Role = models.Role(strings.ToLower(role))
	}

	if dept := c.Query("department_id"); dept != "" {
		id, err := uuid.Parse(dept)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid department_id")
		}
		params.DepartmentID = &id
	}

	if inc := c.Query("include_inactive"); inc == "true" {
		params.IncludeInactive = true
	}

	ctx := requestContext(c)
	users, err := h.users.List(ctx, params)
	if err != nil {
		return err
	}

	items := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		user := u
		items = append(items, adminUserView(&user))
	}

	return c.JSON(fiber.Map{"items": items})
}

// CreateUser creates a new user with specific role.
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var payload dto.UserPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	if payload.Password == nil || len(*payload.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	user, err := h.buildUserFromPayload(payload)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)

	ctx := requestContext(c)
	if err := h.users.Create(ctx, user); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": adminUserView(user)})
}

// UpdateUser updates an existing user.
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var payload dto.UserPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	ctx := requestContext(c)
	user, err := h.users.FindByID(ctx, id)
	if err != nil {
		return err
	}

	updated, err := h.buildUserFromPayload(payload)
	if err != nil {
		return err
	}

	user.FullName = updated.FullName
	user.Role = updated.Role
	user.DepartmentID = updated.DepartmentID
	user.ManagerID = updated.ManagerID
	if payload.IsActive != nil {
		user.IsActive = *payload.IsActive
	}

	if err := h.users.UpdateRole(ctx, user); err != nil {
		return err
	}

	if payload.Password != nil && *payload.Password != "" {
		if len(*payload.Password) < 8 {
			return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*payload.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := h.users.SetPassword(ctx, user.ID, string(hash)); err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{"user": adminUserView(user)})
}

// ListDepartments returns departments.
func (h *AdminHandler) ListDepartments(c *fiber.Ctx) error {
	ctx := requestContext(c)
	deps, err := h.departments.ListDepartments(ctx)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": deps})
}

// CreateDepartment creates department.
func (h *AdminHandler) CreateDepartment(c *fiber.Ctx) error {
	var payload dto.DepartmentPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	dept := &models.Department{
		Name: strings.TrimSpace(payload.Name),
		Code: strings.TrimSpace(payload.Code),
	}
	ctx := requestContext(c)
	if err := h.departments.CreateDepartment(ctx, dept); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"department": dept})
}

// UpdateDepartment updates department.
func (h *AdminHandler) UpdateDepartment(c *fiber.Ctx) error {
	deptID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var payload dto.DepartmentPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	dept := &models.Department{
		ID:   deptID,
		Name: strings.TrimSpace(payload.Name),
		Code: strings.TrimSpace(payload.Code),
	}
	ctx := requestContext(c)
	if err := h.departments.UpdateDepartment(ctx, dept); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"department": dept})
}

// DeleteDepartment deletes department.
func (h *AdminHandler) DeleteDepartment(c *fiber.Ctx) error {
	deptID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	ctx := requestContext(c)
	if err := h.departments.DeleteDepartment(ctx, deptID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListProjects returns projects.
func (h *AdminHandler) ListProjects(c *fiber.Ctx) error {
	var deptID *uuid.UUID
	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid department_id")
		}
		deptID = &id
	}
	ctx := requestContext(c)
	projects, err := h.departments.ListProjects(ctx, deptID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": projects})
}

// CreateProject creates project.
func (h *AdminHandler) CreateProject(c *fiber.Ctx) error {
	var payload dto.ProjectPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	project, err := h.buildProjectFromPayload(payload)
	if err != nil {
		return err
	}
	ctx := requestContext(c)
	if err := h.departments.CreateProject(ctx, project); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"project": project})
}

// UpdateProject updates project.
func (h *AdminHandler) UpdateProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var payload dto.ProjectPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	project, err := h.buildProjectFromPayload(payload)
	if err != nil {
		return err
	}
	project.ID = id
	ctx := requestContext(c)
	if err := h.departments.UpdateProject(ctx, project); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"project": project})
}

// DeleteProject deletes project.
func (h *AdminHandler) DeleteProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	ctx := requestContext(c)
	if err := h.departments.DeleteProject(ctx, id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListCategories returns expense categories.
func (h *AdminHandler) ListCategories(c *fiber.Ctx) error {
	ctx := requestContext(c)
	cats, err := h.references.ListCategories(ctx)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": cats})
}

// CreateCategory creates category.
func (h *AdminHandler) CreateCategory(c *fiber.Ctx) error {
	var payload dto.ExpenseCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	category := &models.ExpenseCategory{
		Name:     strings.TrimSpace(payload.Name),
		Code:     strings.TrimSpace(payload.Code),
		IsActive: payload.IsActive,
	}
	ctx := requestContext(c)
	if err := h.references.UpsertCategory(ctx, category); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"category": category})
}

// UpdateCategory updates category.
func (h *AdminHandler) UpdateCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var payload dto.ExpenseCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	category := &models.ExpenseCategory{
		ID:       id,
		Name:     strings.TrimSpace(payload.Name),
		Code:     strings.TrimSpace(payload.Code),
		IsActive: payload.IsActive,
	}
	ctx := requestContext(c)
	if err := h.references.UpsertCategory(ctx, category); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"category": category})
}

// DeleteCategory deletes category.
func (h *AdminHandler) DeleteCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	ctx := requestContext(c)
	if err := h.references.DeleteCategory(ctx, id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// CreateBudget creates a budget entry.
func (h *AdminHandler) CreateBudget(c *fiber.Ctx) error {
	var payload dto.BudgetPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	budget, err := h.buildBudgetFromPayload(payload)
	if err != nil {
		return err
	}
	ctx := requestContext(c)
	if err := h.budgets.Create(ctx, budget); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"budget": budget})
}

// ListBudgets returns budgets optionally filtered.
func (h *AdminHandler) ListBudgets(c *fiber.Ctx) error {
	var scopeType *models.BudgetScopeType
	switch c.Query("scope_type") {
	case "department":
		t := models.BudgetScopeDepartment
		scopeType = &t
	case "project":
		t := models.BudgetScopeProject
		scopeType = &t
	case "":
	default:
		return fiber.NewError(fiber.StatusBadRequest, "invalid scope_type")
	}

	var scopeID *uuid.UUID
	if raw := c.Query("scope_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid scope_id")
		}
		scopeID = &id
	}

	ctx := requestContext(c)
	items, err := h.budgets.List(ctx, scopeType, scopeID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *AdminHandler) buildUserFromPayload(payload dto.UserPayload) (*models.User, error) {
	email := strings.TrimSpace(strings.ToLower(payload.Email))
	fullName := strings.TrimSpace(payload.FullName)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}
	if fullName == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "full name required")
	}

	role := models.Role(strings.ToLower(payload.Role))
	switch role {
	case models.RoleEmployee, models.RoleManager, models.RoleAccountant, models.RoleAdmin:
	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid role")
	}

	deptID, err := parseUUIDPointer(payload.DepartmentID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid department_id")
	}
	managerID, err := parseUUIDPointer(payload.ManagerID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid manager_id")
	}
	isActive := true
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}

	return &models.User{
		Email:        email,
		FullName:     fullName,
		Role:         role,
		DepartmentID: deptID,
		ManagerID:    managerID,
		IsActive:     isActive,
	}, nil
}

func (h *AdminHandler) buildProjectFromPayload(payload dto.ProjectPayload) (*models.Project, error) {
	name := strings.TrimSpace(payload.Name)
	code := strings.TrimSpace(payload.Code)
	if name == "" || code == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "name and code required")
	}
	deptID, err := parseUUIDPointer(payload.DepartmentID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid department_id")
	}
	return &models.Project{
		Name:         name,
		Code:         code,
		DepartmentID: deptID,
		IsActive:     payload.IsActive,
	}, nil
}

func (h *AdminHandler) buildBudgetFromPayload(payload dto.BudgetPayload) (*models.Budget, error) {
	scopeType := models.BudgetScopeType(strings.ToLower(payload.ScopeType))
	if scopeType != models.BudgetScopeDepartment && scopeType != models.BudgetScopeProject {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid scope_type")
	}
	scopeID, err := uuid.Parse(payload.ScopeID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid scope_id")
	}
	startDate, err := time.Parse("2006-01-02", payload.PeriodStart)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid period_start")
	}
	endDate, err := time.Parse("2006-01-02", payload.PeriodEnd)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid period_end")
	}
	if endDate.Before(startDate) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "period_end must be after start")
	}
	if payload.TotalLimit <= 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "total_limit must be positive")
	}
	return &models.Budget{
		ScopeType:      scopeType,
		ScopeID:        scopeID,
		PeriodStart:    startDate,
		PeriodEnd:      endDate,
		TotalLimit:     payload.TotalLimit,
		Currency:       strings.ToUpper(strings.TrimSpace(payload.Currency)),
		ReservedAmount: 0,
		SpentAmount:    0,
	}, nil
}

func adminUserView(user *models.User) fiber.Map {
	return fiber.Map{
		"id":            user.ID,
		"email":         user.Email,
		"full_name":     user.FullName,
		"role":          user.Role,
		"department_id": user.DepartmentID,
		"manager_id":    user.ManagerID,
		"is_active":     user.IsActive,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	}
}
