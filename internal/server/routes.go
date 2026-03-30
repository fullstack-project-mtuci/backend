package server

import (
	"github.com/gofiber/fiber/v2"
	swagger "github.com/gofiber/swagger"

	_ "backend/docs"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/models"
)

// RegisterRoutes wires HTTP routes with handlers and guards.
func RegisterRoutes(
	app *fiber.App,
	authHandler *handlers.AuthHandler,
	tripHandler *handlers.TripHandler,
	adminHandler *handlers.AdminHandler,
	advanceHandler *handlers.AdvanceHandler,
	expenseHandler *handlers.ExpenseReportHandler,
	receiptHandler *handlers.ReceiptHandler,
	auditHandler *handlers.AuditHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	api := app.Group("/api")
	v1 := api.Group("/v1")
	app.Get("/docs/*", swagger.HandlerDefault)

	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Get("/me", authMiddleware.Handle, authHandler.Me)

	trips := v1.Group("/trip-requests", authMiddleware.Handle)
	trips.Get("/", tripHandler.List)
	trips.Post("/", tripHandler.Create)
	trips.Get("/:id", tripHandler.Get)
	trips.Put("/:id", tripHandler.Update)
	trips.Patch("/:id/status", tripHandler.UpdateStatus)
	trips.Delete("/:id", tripHandler.Delete)

	advance := trips.Group("/:tripId/advance")
	advance.Get("/", advanceHandler.Get)
	advance.Post("/", advanceHandler.Create)
	advance.Patch("/status", advanceHandler.UpdateStatus)

	reportGroup := trips.Group("/:tripId/expense-report")
	reportGroup.Get("/", expenseHandler.GetByTrip)
	reportGroup.Post("/", expenseHandler.Create)

	expReports := v1.Group("/expense-reports", authMiddleware.Handle)
	expReports.Get("/:reportId", expenseHandler.Get)
	expReports.Post("/:reportId/items", expenseHandler.AddItem)
	expReports.Put("/:reportId/items/:itemId", expenseHandler.UpdateItem)
	expReports.Delete("/:reportId/items/:itemId", expenseHandler.DeleteItem)
	expReports.Patch("/:reportId/status", expenseHandler.UpdateStatus)

	receipts := v1.Group("/receipts", authMiddleware.Handle)
	receipts.Get("/", receiptHandler.List)
	receipts.Post("/", receiptHandler.Upload)

	admin := v1.Group("/admin", authMiddleware.Handle, middleware.RequireRoles(models.RoleAdmin))
	admin.Get("/users", adminHandler.ListUsers)
	admin.Post("/users", adminHandler.CreateUser)
	admin.Put("/users/:id", adminHandler.UpdateUser)

	admin.Get("/departments", adminHandler.ListDepartments)
	admin.Post("/departments", adminHandler.CreateDepartment)
	admin.Put("/departments/:id", adminHandler.UpdateDepartment)
	admin.Delete("/departments/:id", adminHandler.DeleteDepartment)

	admin.Get("/projects", adminHandler.ListProjects)
	admin.Post("/projects", adminHandler.CreateProject)
	admin.Put("/projects/:id", adminHandler.UpdateProject)
	admin.Delete("/projects/:id", adminHandler.DeleteProject)

	admin.Get("/categories", adminHandler.ListCategories)
	admin.Post("/categories", adminHandler.CreateCategory)
	admin.Put("/categories/:id", adminHandler.UpdateCategory)
	admin.Delete("/categories/:id", adminHandler.DeleteCategory)

	admin.Post("/budgets", adminHandler.CreateBudget)
	admin.Get("/budgets", adminHandler.ListBudgets)

	audit := v1.Group("/audit", authMiddleware.Handle)
	audit.Get("/:entityType/:entityId/approvals", auditHandler.ListApprovals)
	audit.Get("/:entityType/:entityId/logs", auditHandler.ListAuditLogs)
}
