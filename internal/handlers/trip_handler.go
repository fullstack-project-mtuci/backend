package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
)

// TripHandler manages TripRequest CRUD operations.
type TripHandler struct {
	trips   *repositories.TripRepository
	budgets *repositories.BudgetRepository
	users   *repositories.UserRepository
	cfg     *config.Config
	logger  *services.WorkflowLogger
}

// NewTripHandler constructs a TripHandler.
func NewTripHandler(trips *repositories.TripRepository, budgets *repositories.BudgetRepository, users *repositories.UserRepository, cfg *config.Config, logger *services.WorkflowLogger) *TripHandler {
	return &TripHandler{
		trips:   trips,
		budgets: budgets,
		users:   users,
		cfg:     cfg,
		logger:  logger,
	}
}

// Create handles POST /trip-requests.
func (h *TripHandler) Create(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	var payload dto.TripPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	trip, err := h.buildTripFromPayload(user.ID, payload)
	if err != nil {
		return err
	}

	ctx := requestContext(c)
	if err := h.trips.Create(ctx, trip); err != nil {
		return err
	}

	if h.logger != nil {
		if err := h.logger.Audit(ctx, user.ID, "trip_create", "trip_request", trip.ID, nil, trip, c.IP(), c.Get("User-Agent")); err != nil {
			log.Printf("audit failed: %v", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"trip": trip})
}

// List handles GET /trip-requests.
func (h *TripHandler) List(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	ctx := requestContext(c)
	var (
		trips []models.TripRequest
		err   error
	)
	if user.Role == models.RoleEmployee {
		trips, err = h.trips.ListForEmployee(ctx, user.ID)
	} else {
		trips, err = h.trips.ListAll(ctx)
	}
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": trips})
}

// Get handles GET /trip-requests/:id.
func (h *TripHandler) Get(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.fetchTrip(c)
	if err != nil {
		return err
	}

	if user.Role == models.RoleEmployee && trip.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}

	return c.JSON(fiber.Map{"trip": trip})
}

// Update handles PUT /trip-requests/:id.
func (h *TripHandler) Update(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.fetchTrip(c)
	if err != nil {
		return err
	}

	if trip.Status != models.TripStatusDraft && user.Role == models.RoleEmployee {
		return fiber.NewError(fiber.StatusBadRequest, "only draft trips can be edited")
	}

	if user.Role == models.RoleEmployee && trip.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}

	var payload dto.TripPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	before := *trip
	updatedTrip, err := h.buildTripFromPayload(trip.EmployeeID, payload)
	if err != nil {
		return err
	}
	updatedTrip.ID = trip.ID
	updatedTrip.Status = trip.Status
	updatedTrip.BudgetID = trip.BudgetID

	ctx := requestContext(c)
	if err := h.trips.Update(ctx, updatedTrip); err != nil {
		return err
	}

	if h.logger != nil {
		if err := h.logger.Audit(ctx, user.ID, "trip_update", "trip_request", updatedTrip.ID, &before, updatedTrip, c.IP(), c.Get("User-Agent")); err != nil {
			log.Printf("audit failed: %v", err)
		}
	}

	return c.JSON(fiber.Map{"trip": updatedTrip})
}

// UpdateStatus handles PATCH /trip-requests/:id/status.
func (h *TripHandler) UpdateStatus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.fetchTrip(c)
	if err != nil {
		return err
	}
	before := *trip

	var payload dto.StatusPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	payload.Status = strings.ToLower(payload.Status)
	newStatus := models.TripStatus(payload.Status)

	if !canChangeTripStatus(trip.Status, newStatus) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("transition from %s to %s is not allowed", trip.Status, newStatus))
	}

	if err := h.authorizeStatusChange(user.Role, newStatus); err != nil {
		return err
	}

	ctx := requestContext(c)
	update := repositories.TripStatusUpdate{}
	now := time.Now()
	var warnings []string

	switch newStatus {
	case models.TripStatusSubmitted:
		update.SubmittedAt = &now
		budget, warning, err := h.reserveBudget(ctx, trip)
		if err != nil {
			return err
		}
		if budget != nil {
			update.BudgetID = &budget.ID
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	case models.TripStatusManagerApproved:
		update.ApprovedAt = &now
	case models.TripStatusAccountantReview:
		// no-op
	case models.TripStatusManagerRejected, models.TripStatusCancelled:
		update.RejectedAt = &now
		if err := h.releaseBudget(ctx, trip); err != nil {
			return err
		}
	case models.TripStatusApproved:
		update.ApprovedAt = &now
		if err := h.finalizeBudget(ctx, trip); err != nil {
			return err
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported status")
	}

	trip, err = h.trips.UpdateStatus(ctx, trip.ID, newStatus, update)
	if err != nil {
		return err
	}

	if h.logger != nil {
		if err := h.logger.Audit(ctx, user.ID, "trip_status_change", "trip_request", trip.ID, &before, trip, c.IP(), c.Get("User-Agent")); err != nil {
			log.Printf("audit failed: %v", err)
		}
		if err := h.logger.Approval(ctx, "trip_request", trip.ID, string(newStatus), user.ID, payload.Comment); err != nil {
			log.Printf("failed to log trip approval: %v", err)
		}
	}

	resp := fiber.Map{"trip": trip}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	return c.JSON(resp)
}

// Delete handles DELETE /trip-requests/:id.
func (h *TripHandler) Delete(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.fetchTrip(c)
	if err != nil {
		return err
	}

	if user.Role == models.RoleEmployee && trip.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}

	ctx := requestContext(c)
	if err := h.trips.SoftDelete(ctx, trip.ID); err != nil {
		return h.repoError(err)
	}
	if h.logger != nil {
		if err := h.logger.Audit(ctx, user.ID, "trip_delete", "trip_request", trip.ID, trip, nil, c.IP(), c.Get("User-Agent")); err != nil {
			log.Printf("audit failed: %v", err)
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *TripHandler) fetchTrip(c *fiber.Ctx) (*models.TripRequest, error) {
	idParam := c.Params("id")
	if idParam == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "missing id")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	ctx := requestContext(c)
	trip, err := h.trips.GetByID(ctx, id)
	if err != nil {
		return nil, h.repoError(err)
	}
	return trip, nil
}

func (h *TripHandler) buildTripFromPayload(employeeID uuid.UUID, payload dto.TripPayload) (*models.TripRequest, error) {
	startDate, err := time.Parse("2006-01-02", strings.TrimSpace(payload.StartDate))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid start_date")
	}
	endDate, err := time.Parse("2006-01-02", strings.TrimSpace(payload.EndDate))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid end_date")
	}
	if endDate.Before(startDate) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "end_date must be after start_date")
	}

	projectID, err := parseUUIDPointer(payload.ProjectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid project_id")
	}

	trip := &models.TripRequest{
		EmployeeID:            employeeID,
		ProjectID:             projectID,
		DestinationCity:       strings.TrimSpace(payload.DestinationCity),
		DestinationCountry:    strings.TrimSpace(payload.DestinationCountry),
		Purpose:               strings.TrimSpace(payload.Purpose),
		Comment:               strings.TrimSpace(payload.Comment),
		StartDate:             startDate,
		EndDate:               endDate,
		PlannedTransport:      payload.PlannedTransport,
		PlannedHotel:          payload.PlannedHotel,
		PlannedDailyAllowance: payload.PlannedDailyAllowance,
		PlannedOther:          payload.PlannedOther,
		Currency:              strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Status:                models.TripStatusDraft,
	}
	trip.PlannedTotal = trip.PlannedTransport + trip.PlannedHotel + trip.PlannedDailyAllowance + trip.PlannedOther

	if trip.DestinationCity == "" || trip.DestinationCountry == "" || trip.Purpose == "" || trip.Currency == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "city, country, purpose and currency are required")
	}

	return trip, nil
}

func parseUUIDPointer(value *string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (h *TripHandler) repoError(err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return fiber.ErrNotFound
	}
	return err
}

func (h *TripHandler) authorizeStatusChange(role models.Role, status models.TripStatus) error {
	switch status {
	case models.TripStatusSubmitted, models.TripStatusCancelled:
		if role != models.RoleEmployee && role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.TripStatusManagerApproved, models.TripStatusManagerRejected:
		if role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.TripStatusAccountantReview, models.TripStatusApproved:
		if role != models.RoleAccountant && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	default:
		return fiber.ErrForbidden
	}
	return nil
}

func canChangeTripStatus(current, next models.TripStatus) bool {
	switch current {
	case models.TripStatusDraft:
		return next == models.TripStatusSubmitted || next == models.TripStatusCancelled
	case models.TripStatusSubmitted:
		return next == models.TripStatusManagerApproved || next == models.TripStatusManagerRejected || next == models.TripStatusCancelled
	case models.TripStatusManagerApproved:
		return next == models.TripStatusAccountantReview || next == models.TripStatusManagerRejected
	case models.TripStatusAccountantReview:
		return next == models.TripStatusApproved || next == models.TripStatusManagerRejected
	case models.TripStatusManagerRejected:
		return next == models.TripStatusDraft || next == models.TripStatusCancelled
	default:
		return false
	}
}

func (h *TripHandler) reserveBudget(ctx context.Context, trip *models.TripRequest) (*models.Budget, string, error) {
	budget, err := h.findBudget(ctx, trip)
	if err != nil {
		return nil, "", err
	}

	available := budget.TotalLimit - budget.ReservedAmount
	shortfall := trip.PlannedTotal - available
	warning := ""
	if shortfall > 0 {
		if h.cfg.BudgetLimitMode == "hard" {
			return nil, "", fiber.NewError(fiber.StatusUnprocessableEntity, "budget limit exceeded")
		}
		warning = fmt.Sprintf("budget limit exceeded by %.2f %s", shortfall, budget.Currency)
	}

	if err := h.budgets.AdjustReserved(ctx, budget.ID, trip.PlannedTotal); err != nil {
		return nil, "", err
	}

	return budget, warning, nil
}

func (h *TripHandler) releaseBudget(ctx context.Context, trip *models.TripRequest) error {
	if trip.BudgetID == nil {
		return nil
	}
	return h.budgets.AdjustReserved(ctx, *trip.BudgetID, -trip.PlannedTotal)
}

func (h *TripHandler) finalizeBudget(ctx context.Context, trip *models.TripRequest) error {
	if trip.BudgetID == nil {
		return nil
	}
	return h.budgets.Consume(ctx, *trip.BudgetID, trip.PlannedTotal)
}

func (h *TripHandler) findBudget(ctx context.Context, trip *models.TripRequest) (*models.Budget, error) {
	if trip.ProjectID != nil {
		budget, err := h.budgets.FindActiveBudget(ctx, models.BudgetScopeProject, *trip.ProjectID, trip.StartDate)
		if err == nil {
			return budget, nil
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
	}

	employee, err := h.users.FindByID(ctx, trip.EmployeeID)
	if err != nil {
		return nil, err
	}

	if employee.DepartmentID != nil {
		budget, err := h.budgets.FindActiveBudget(ctx, models.BudgetScopeDepartment, *employee.DepartmentID, trip.StartDate)
		if err == nil {
			return budget, nil
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
	}

	return nil, fiber.NewError(fiber.StatusUnprocessableEntity, "budget not configured for project or department")
}
