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

// ExpenseReportHandler handles reports and items.
type ExpenseReportHandler struct {
	trips    *repositories.TripRepository
	reports  *repositories.ExpenseReportRepository
	items    *repositories.ExpenseItemRepository
	advances *repositories.AdvanceRepository
	receipts *repositories.ReceiptRepository
	logger   *services.WorkflowLogger
}

// NewExpenseReportHandler constructs handler.
func NewExpenseReportHandler(
	trips *repositories.TripRepository,
	reports *repositories.ExpenseReportRepository,
	items *repositories.ExpenseItemRepository,
	advances *repositories.AdvanceRepository,
	receipts *repositories.ReceiptRepository,
	logger *services.WorkflowLogger,
) *ExpenseReportHandler {
	return &ExpenseReportHandler{
		trips:    trips,
		reports:  reports,
		items:    items,
		advances: advances,
		receipts: receipts,
		logger:   logger,
	}
}

// Create godoc
// @Summary Create expense report
// @Tags ExpenseReports
// @Produce json
// @Security BearerAuth
// @Param tripId path string true "Trip ID"
// @Success 201 {object} map[string]interface{}
// @Router /trip-requests/{tripId}/expense-report [post]
func (h *ExpenseReportHandler) Create(c *fiber.Ctx) error {
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

	ctx := requestContext(c)
	if existing, err := h.reports.FindByTrip(ctx, trip.ID); err == nil {
		return fiber.NewError(fiber.StatusConflict, "report already exists with id "+existing.ID.String())
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return err
	}

	advanceAmount := 0.0
	if adv, err := h.advances.FindByTrip(ctx, trip.ID); err == nil && (adv.Status == models.AdvanceStatusApproved || adv.Status == models.AdvanceStatusPaid) {
		advanceAmount = adv.ApprovedAmount
	}

	report := &models.ExpenseReport{
		TripRequestID: trip.ID,
		EmployeeID:    trip.EmployeeID,
		AdvanceAmount: advanceAmount,
		TotalExpenses: 0,
		BalanceAmount: advanceAmount,
		Currency:      trip.Currency,
		Status:        models.ExpenseReportDraft,
	}

	if err := h.reports.Create(ctx, report); err != nil {
		return err
	}

	h.logAudit(ctx, user.ID, "expense_report_create", report.ID, nil, report, c)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"report": report})
}

// GetByTrip godoc
// @Summary Get expense report for trip
// @Tags ExpenseReports
// @Produce json
// @Security BearerAuth
// @Param tripId path string true "Trip ID"
// @Success 200 {object} map[string]interface{}
// @Router /trip-requests/{tripId}/expense-report [get]
func (h *ExpenseReportHandler) GetByTrip(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	trip, err := h.loadTrip(c)
	if err != nil {
		return err
	}

	ctx := requestContext(c)
	report, err := h.reports.FindByTrip(ctx, trip.ID)
	if err != nil {
		return h.repoError(err)
	}

	if user.Role == models.RoleEmployee && report.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}

	items, err := h.items.ListByReport(ctx, report.ID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"report": report, "items": items})
}

// Get godoc
// @Summary Get expense report by ID
// @Tags ExpenseReports
// @Produce json
// @Security BearerAuth
// @Param reportId path string true "Report ID"
// @Success 200 {object} map[string]interface{}
// @Router /expense-reports/{reportId} [get]
func (h *ExpenseReportHandler) Get(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	report, err := h.loadReport(c)
	if err != nil {
		return err
	}
	if user.Role == models.RoleEmployee && report.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}
	ctx := requestContext(c)
	items, err := h.items.ListByReport(ctx, report.ID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"report": report, "items": items})
}

// AddItem godoc
// @Summary Add expense item
// @Tags ExpenseReports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param reportId path string true "Report ID"
// @Param payload body dto.ExpenseItemPayload true "Expense item payload"
// @Success 201 {object} map[string]interface{}
// @Router /expense-reports/{reportId}/items [post]
func (h *ExpenseReportHandler) AddItem(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	report, err := h.loadReport(c)
	if err != nil {
		return err
	}
	if err := h.ensureEditable(report, user); err != nil {
		return err
	}

	ctx := requestContext(c)
	var payload dto.ExpenseItemPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	item, err := h.buildExpenseItem(ctx, report, payload)
	if err != nil {
		return err
	}

	if err := h.items.Create(ctx, item); err != nil {
		return err
	}
	if err := h.recalculateReport(ctx, report.ID); err != nil {
		return err
	}
	h.logAudit(ctx, user.ID, "expense_item_add", report.ID, nil, item, c)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"item": item})
}

// UpdateItem godoc
// @Summary Update expense item
// @Tags ExpenseReports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param reportId path string true "Report ID"
// @Param itemId path string true "Item ID"
// @Param payload body dto.ExpenseItemPayload true "Expense item payload"
// @Success 200 {object} map[string]interface{}
// @Router /expense-reports/{reportId}/items/{itemId} [put]
func (h *ExpenseReportHandler) UpdateItem(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	report, err := h.loadReport(c)
	if err != nil {
		return err
	}
	if err := h.ensureEditable(report, user); err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Params("itemId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid item id")
	}
	ctx := requestContext(c)
	item, err := h.items.Get(ctx, itemID)
	if err != nil {
		return h.repoError(err)
	}
	if item.ExpenseReportID != report.ID {
		return fiber.ErrForbidden
	}

	var payload dto.ExpenseItemPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	updated, err := h.buildExpenseItem(ctx, report, payload)
	if err != nil {
		return err
	}
	updated.ID = item.ID

	if err := h.items.Update(ctx, updated); err != nil {
		return err
	}
	if err := h.recalculateReport(ctx, report.ID); err != nil {
		return err
	}

	h.logAudit(ctx, user.ID, "expense_item_update", report.ID, item, updated, c)

	return c.JSON(fiber.Map{"item": updated})
}

// DeleteItem godoc
// @Summary Delete expense item
// @Tags ExpenseReports
// @Security BearerAuth
// @Param reportId path string true "Report ID"
// @Param itemId path string true "Item ID"
// @Success 204
// @Router /expense-reports/{reportId}/items/{itemId} [delete]
func (h *ExpenseReportHandler) DeleteItem(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	report, err := h.loadReport(c)
	if err != nil {
		return err
	}
	if err := h.ensureEditable(report, user); err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Params("itemId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid item id")
	}
	ctx := requestContext(c)
	item, err := h.items.Get(ctx, itemID)
	if err != nil {
		return h.repoError(err)
	}
	if item.ExpenseReportID != report.ID {
		return fiber.ErrForbidden
	}

	if err := h.items.Delete(ctx, item.ID); err != nil {
		return err
	}
	if err := h.recalculateReport(ctx, report.ID); err != nil {
		return err
	}
	h.logAudit(ctx, user.ID, "expense_item_delete", report.ID, item, nil, c)
	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateStatus godoc
// @Summary Change expense report status
// @Tags ExpenseReports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param reportId path string true "Report ID"
// @Param payload body dto.ExpenseReportStatusPayload true "Status payload"
// @Success 200 {object} map[string]interface{}
// @Router /expense-reports/{reportId}/status [patch]
func (h *ExpenseReportHandler) UpdateStatus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}
	report, err := h.loadReport(c)
	if err != nil {
		return err
	}

	var payload dto.ExpenseReportStatusPayload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	next := models.ExpenseReportStatus(strings.ToLower(payload.Status))
	if !canReportStatusChange(report.Status, next) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status transition")
	}
	if err := authorizeReportStatus(user.Role, next); err != nil {
		return err
	}

	now := time.Now()
	switch next {
	case models.ExpenseReportSubmitted:
		report.SubmittedAt = &now
	case models.ExpenseReportManagerReview:
		report.ReviewedAt = &now
	case models.ExpenseReportApproved:
		report.ClosedAt = &now
	case models.ExpenseReportRejected, models.ExpenseReportNeedsRevision:
		// allow edits again
	}
	before := *report

	report.Status = next
	ctx := requestContext(c)
	if err := h.reports.Update(ctx, report); err != nil {
		return err
	}
	if h.logger != nil {
		h.logAudit(ctx, user.ID, "expense_report_status", report.ID, &before, report, c)
		if err := h.logger.Approval(ctx, "expense_report", report.ID, string(next), user.ID, payload.Comment); err != nil {
			log.Printf("failed to log report status: %v", err)
		}
	}
	return c.JSON(fiber.Map{"report": report})
}

func (h *ExpenseReportHandler) ensureEditable(report *models.ExpenseReport, user *models.User) error {
	if user.Role == models.RoleEmployee && report.EmployeeID != user.ID {
		return fiber.ErrForbidden
	}
	if report.Status != models.ExpenseReportDraft && report.Status != models.ExpenseReportNeedsRevision {
		return fiber.NewError(fiber.StatusBadRequest, "report is not editable")
	}
	return nil
}

func (h *ExpenseReportHandler) loadTrip(c *fiber.Ctx) (*models.TripRequest, error) {
	tripID, err := uuid.Parse(c.Params("tripId"))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid trip id")
	}
	ctx := requestContext(c)
	return h.trips.GetByID(ctx, tripID)
}

func (h *ExpenseReportHandler) loadReport(c *fiber.Ctx) (*models.ExpenseReport, error) {
	reportID, err := uuid.Parse(c.Params("reportId"))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid report id")
	}
	ctx := requestContext(c)
	return h.reports.Get(ctx, reportID)
}

func (h *ExpenseReportHandler) buildExpenseItem(ctx context.Context, report *models.ExpenseReport, payload dto.ExpenseItemPayload) (*models.ExpenseItem, error) {
	if payload.Amount <= 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "amount must be positive")
	}
	expenseDate, err := time.Parse("2006-01-02", strings.TrimSpace(payload.ExpenseDate))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid expense_date")
	}
	category := strings.TrimSpace(payload.Category)
	if category == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "category is required")
	}
	currency := strings.ToUpper(strings.TrimSpace(payload.Currency))
	if currency == "" {
		currency = report.Currency
	}

	var receiptID *uuid.UUID
	if payload.ReceiptFileID != nil {
		id, err := uuid.Parse(*payload.ReceiptFileID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid receipt_file_id")
		}
		receiptID = &id
		receipt, err := h.receipts.GetFile(ctx, *receiptID)
		if err != nil {
			return nil, h.repoError(err)
		}
		if receipt.UploadedBy != report.EmployeeID {
			return nil, fiber.ErrForbidden
		}
	}

	item := &models.ExpenseItem{
		ExpenseReportID: report.ID,
		Category:        category,
		ExpenseDate:     expenseDate,
		VendorName:      payload.VendorName,
		Amount:          payload.Amount,
		Currency:        currency,
		TaxAmount:       payload.TaxAmount,
		Description:     payload.Description,
		ReceiptFileID:   receiptID,
		Source:          models.ExpenseSource(payload.Source),
		Status:          models.ExpenseItemDraft,
	}
	if item.Source == "" {
		item.Source = models.ExpenseSourceManual
	}
	if payload.Status != nil && *payload.Status != "" {
		item.Status = models.ExpenseItemStatus(strings.ToLower(*payload.Status))
	}
	return item, nil
}

func (h *ExpenseReportHandler) recalculateReport(ctx context.Context, reportID uuid.UUID) error {
	items, err := h.items.ListByReport(ctx, reportID)
	if err != nil {
		return err
	}
	total := 0.0
	for _, item := range items {
		total += item.Amount
	}
	report, err := h.reports.Get(ctx, reportID)
	if err != nil {
		return err
	}
	report.TotalExpenses = total
	report.BalanceAmount = report.AdvanceAmount - total
	return h.reports.Update(ctx, report)
}

func canReportStatusChange(current, next models.ExpenseReportStatus) bool {
	switch current {
	case models.ExpenseReportDraft, models.ExpenseReportNeedsRevision:
		return next == models.ExpenseReportSubmitted
	case models.ExpenseReportSubmitted:
		return next == models.ExpenseReportManagerReview
	case models.ExpenseReportManagerReview:
		return next == models.ExpenseReportAccountantReview
	case models.ExpenseReportAccountantReview:
		return next == models.ExpenseReportApproved || next == models.ExpenseReportRejected || next == models.ExpenseReportNeedsRevision
	case models.ExpenseReportApproved:
		return next == models.ExpenseReportClosed
	default:
		return false
	}
}

func authorizeReportStatus(role models.Role, status models.ExpenseReportStatus) error {
	switch status {
	case models.ExpenseReportSubmitted:
		if role != models.RoleEmployee && role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.ExpenseReportManagerReview:
		if role != models.RoleManager && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	case models.ExpenseReportAccountantReview, models.ExpenseReportApproved, models.ExpenseReportRejected, models.ExpenseReportNeedsRevision, models.ExpenseReportClosed:
		if role != models.RoleAccountant && role != models.RoleAdmin {
			return fiber.ErrForbidden
		}
	default:
		return fiber.ErrForbidden
	}
	return nil
}

func (h *ExpenseReportHandler) repoError(err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return fiber.ErrNotFound
	}
	return err
}

func (h *ExpenseReportHandler) logAudit(ctx context.Context, actorID uuid.UUID, action string, entityID uuid.UUID, before, after interface{}, c *fiber.Ctx) {
	if h.logger == nil {
		return
	}
	if err := h.logger.Audit(ctx, actorID, action, "expense_report", entityID, before, after, c.IP(), c.Get("User-Agent")); err != nil {
		log.Printf("audit failed: %v", err)
	}
}
