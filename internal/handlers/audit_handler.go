package handlers

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
)

// AuditHandler exposes approval and audit logs.
type AuditHandler struct {
	approvals *repositories.ApprovalRepository
	audits    *repositories.AuditRepository
	trips     *repositories.TripRepository
	advances  *repositories.AdvanceRepository
	reports   *repositories.ExpenseReportRepository
}

// NewAuditHandler constructs handler.
func NewAuditHandler(
	approvals *repositories.ApprovalRepository,
	audits *repositories.AuditRepository,
	trips *repositories.TripRepository,
	advances *repositories.AdvanceRepository,
	reports *repositories.ExpenseReportRepository,
) *AuditHandler {
	return &AuditHandler{
		approvals: approvals,
		audits:    audits,
		trips:     trips,
		advances:  advances,
		reports:   reports,
	}
}

// ListApprovals godoc
// @Summary List approval actions
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param entityType path string true "Entity type" Enums(trip_request,advance_request,expense_report)
// @Param entityId path string true "Entity ID"
// @Success 200 {object} map[string]interface{}
// @Router /audit/{entityType}/{entityId}/approvals [get]
func (h *AuditHandler) ListApprovals(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	entityType, entityID, err := h.parseEntity(c)
	if err != nil {
		return err
	}

	ctx := requestContext(c)
	if err := h.authorize(ctx, user, entityType, entityID); err != nil {
		return err
	}

	actions, err := h.approvals.ListActions(ctx, entityType, entityID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": actions})
}

// ListAuditLogs godoc
// @Summary List audit log entries
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param entityType path string true "Entity type" Enums(trip_request,advance_request,expense_report)
// @Param entityId path string true "Entity ID"
// @Success 200 {object} map[string]interface{}
// @Router /audit/{entityType}/{entityId}/logs [get]
func (h *AuditHandler) ListAuditLogs(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	entityType, entityID, err := h.parseEntity(c)
	if err != nil {
		return err
	}

	ctx := requestContext(c)
	if err := h.authorize(ctx, user, entityType, entityID); err != nil {
		return err
	}

	logs, err := h.audits.ListLogs(ctx, entityType, entityID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"items": logs})
}

func (h *AuditHandler) parseEntity(c *fiber.Ctx) (string, uuid.UUID, error) {
	entityType := strings.ToLower(c.Params("entityType"))
	id, err := uuid.Parse(c.Params("entityId"))
	if err != nil {
		return "", uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	switch entityType {
	case "trip_request", "advance_request", "expense_report":
		return entityType, id, nil
	default:
		return "", uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "unsupported entity type")
	}
}

func (h *AuditHandler) authorize(ctx context.Context, user *models.User, entityType string, entityID uuid.UUID) error {
	if user.Role == models.RoleAdmin || user.Role == models.RoleManager || user.Role == models.RoleAccountant {
		return nil
	}

	switch entityType {
	case "trip_request":
		trip, err := h.trips.GetByID(ctx, entityID)
		if err != nil {
			return fiber.ErrNotFound
		}
		if trip.EmployeeID != user.ID {
			return fiber.ErrForbidden
		}
	case "advance_request":
		adv, err := h.advances.Get(ctx, entityID)
		if err != nil {
			return fiber.ErrNotFound
		}
		trip, err := h.trips.GetByID(ctx, adv.TripRequestID)
		if err != nil {
			return fiber.ErrNotFound
		}
		if trip.EmployeeID != user.ID {
			return fiber.ErrForbidden
		}
	case "expense_report":
		report, err := h.reports.Get(ctx, entityID)
		if err != nil {
			return fiber.ErrNotFound
		}
		if report.EmployeeID != user.ID {
			return fiber.ErrForbidden
		}
	default:
		return fiber.ErrForbidden
	}
	return nil
}
