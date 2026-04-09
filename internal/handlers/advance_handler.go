package handlers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"backend/internal/dto"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/services"
)

// AdvanceHandler manages advance requests workflow.
type AdvanceHandler struct {
	trips    *repositories.TripRepository
	advances *repositories.AdvanceRepository
	logger   *services.WorkflowLogger
}

// NewAdvanceHandler constructs AdvanceHandler.
func NewAdvanceHandler(trips *repositories.TripRepository, advances *repositories.AdvanceRepository, logger *services.WorkflowLogger) *AdvanceHandler {
	return &AdvanceHandler{trips: trips, advances: advances, logger: logger}
}

// Get godoc
// @Summary Get advance for trip
// @Tags Advances
// @Produce json
// @Security BearerAuth
// @Param tripId path string true "Trip ID"
// @Success 200 {object} map[string]interface{}
// @Router /trip-requests/{tripId}/advance [get]
func (h *AdvanceHandler) Get(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.loadTrip(c)
	if err != nil {
		return err
	}

	if user.Role == models.RoleEmployee && trip.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}

	ctx := requestContext(c)
	adv, err := h.advances.FindByTrip(ctx, trip.ID)
	if err != nil {
		return h.repoError(err)
	}

	return c.JSON(fiber.Map{"advance": adv})
}

// Create godoc
// @Summary Create or update draft advance
// @Tags Advances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tripId path string true "Trip ID"
// @Param payload body dto.AdvancePayload true "Advance payload"
// @Success 201 {object} map[string]interface{}
// @Router /trip-requests/{tripId}/advance [post]
func (h *AdvanceHandler) Create(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.loadTrip(c)
	if err != nil {
		return err
	}

	if trip.EmployeeID != user.ID && user.Role != models.RoleAdmin {
		return fiber.ErrForbidden
	}

	if trip.Status != models.TripStatusManagerApproved && trip.Status != models.TripStatusApproved {
		return fiber.NewError(fiber.StatusBadRequest, "advance allowed only for approved trips")
	}

	var payload dto.AdvancePayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	if payload.RequestedAmount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "requested_amount must be positive")
	}

	currency := strings.ToUpper(strings.TrimSpace(payload.Currency))
	if currency == "" {
		currency = trip.Currency
	}

	ctx := requestContext(c)
	existing, err := h.advances.FindByTrip(ctx, trip.ID)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return err
	}

	if existing != nil {
		before := *existing
		if existing.Status != models.AdvanceStatusDraft {
			return fiber.NewError(fiber.StatusConflict, "advance already submitted")
		}
		existing.RequestedAmount = payload.RequestedAmount
		existing.Currency = currency
		existing.Comment = payload.Comment
		if err := h.advances.Update(ctx, existing); err != nil {
			return err
		}
		h.logAudit(ctx, user.ID, "advance_update", existing.ID, &before, existing, c)
		return c.JSON(fiber.Map{"advance": existing})
	}

	advance := &models.AdvanceRequest{
		TripRequestID:   trip.ID,
		RequestedAmount: payload.RequestedAmount,
		ApprovedAmount:  payload.RequestedAmount,
		Currency:        currency,
		Status:          models.AdvanceStatusDraft,
		Comment:         payload.Comment,
	}

	if err := h.advances.Create(ctx, advance); err != nil {
		return err
	}
	h.logAudit(ctx, user.ID, "advance_create", advance.ID, nil, advance, c)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"advance": advance})
}

// UpdateStatus godoc
// @Summary Change advance status
// @Tags Advances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tripId path string true "Trip ID"
// @Param payload body dto.StatusPayload true "Status payload"
// @Success 200 {object} map[string]interface{}
// @Router /trip-requests/{tripId}/advance/status [patch]
func (h *AdvanceHandler) UpdateStatus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	trip, err := h.loadTrip(c)
	if err != nil {
		return err
	}

	ctx := requestContext(c)
	advance, err := h.advances.FindByTrip(ctx, trip.ID)
	if err != nil {
		return h.repoError(err)
	}

	var payload dto.StatusPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	next := models.AdvanceStatus(strings.ToLower(payload.Status))
	if !canAdvanceStatusChange(advance.Status, next) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status transition")
	}

	if err := authorizeAdvanceStatus(user.Role, next); err != nil {
		return err
	}

	now := time.Now()
	switch next {
	case models.AdvanceStatusSubmitted:
		advance.SubmittedAt = &now
	case models.AdvanceStatusApproved, models.AdvanceStatusRejected, models.AdvanceStatusPaid:
		advance.ProcessedAt = &now
	}
	advance.Status = next

	if err := h.advances.Update(ctx, advance); err != nil {
		return err
	}

	if h.logger != nil {
		if err := h.logger.Approval(ctx, "advance_request", advance.ID, string(next), user.ID, payload.Comment); err != nil {
			log.Printf("failed to log advance approval: %v", err)
		}
	}

	return c.JSON(fiber.Map{"advance": advance})
}

func (h *AdvanceHandler) logAudit(ctx context.Context, actorID uuid.UUID, action string, advanceID uuid.UUID, before, after interface{}, c *fiber.Ctx) {
	if h.logger == nil {
		return
	}
	if err := h.logger.Audit(ctx, actorID, action, "advance_request", advanceID, before, after, c.IP(), c.Get("User-Agent")); err != nil {
		log.Printf("audit failed: %v", err)
	}
}

func (h *AdvanceHandler) loadTrip(c *fiber.Ctx) (*models.TripRequest, error) {
	tripID, err := uuid.Parse(c.Params("tripId"))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid trip id")
	}
	ctx := requestContext(c)
	return h.trips.GetByID(ctx, tripID)
}

func (h *AdvanceHandler) repoError(err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return fiber.ErrNotFound
	}
	return err
}

func canAdvanceStatusChange(current, next models.AdvanceStatus) bool {
	switch current {
	case models.AdvanceStatusDraft:
		return next == models.AdvanceStatusSubmitted
	case models.AdvanceStatusSubmitted:
		return next == models.AdvanceStatusApproved || next == models.AdvanceStatusRejected
	case models.AdvanceStatusApproved:
		return next == models.AdvanceStatusPaid
	default:
		return false
	}
}

func authorizeAdvanceStatus(role models.Role, status models.AdvanceStatus) error {
	switch status {
	case models.AdvanceStatusSubmitted:
		if role != models.RoleEmployee && role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.AdvanceStatusApproved:
		if role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.AdvanceStatusRejected, models.AdvanceStatusPaid:
		if role != models.RoleAccountant && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	default:
		return fiber.ErrForbidden
	}
	return nil
}
