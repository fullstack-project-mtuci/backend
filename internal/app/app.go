package app

import (
	"context"
	"time"

	"go.uber.org/fx"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/ocr"
	"backend/internal/repositories"
	"backend/internal/server"
	"backend/internal/services"
	"backend/internal/storage"
	"backend/internal/tokens"
)

// Start bootstraps the application using Fx and blocks until termination.
func Start(ctx context.Context) error {
	application := fx.New(
		fx.Provide(
			config.Load,
			server.NewFiberApp,
			newTokenManager,
			newStorageClient,
			newOCRAdapter,
			database.NewPool,
			repositories.NewUserRepository,
			repositories.NewTripRepository,
			repositories.NewBudgetRepository,
			repositories.NewDepartmentRepository,
			repositories.NewReferenceRepository,
			repositories.NewAdvanceRepository,
			repositories.NewExpenseReportRepository,
			repositories.NewExpenseItemRepository,
			repositories.NewReceiptRepository,
			repositories.NewApprovalRepository,
			repositories.NewAuditRepository,
			middleware.NewAuthMiddleware,
			handlers.NewAuthHandler,
			handlers.NewTripHandler,
			handlers.NewAdminHandler,
			handlers.NewAdvanceHandler,
			handlers.NewExpenseReportHandler,
			handlers.NewReceiptHandler,
			handlers.NewReferenceHandler,
			handlers.NewAuditHandler,
			services.NewWorkflowLogger,
		),
		fx.Invoke(
			server.RegisterRoutes,
			server.RegisterFiberLifecycle,
			server.RegisterDatabaseLifecycle,
		),
	)

	go func() {
		<-ctx.Done()
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = application.Stop(stopCtx)
	}()

	application.Run()
	return nil
}

func newTokenManager(cfg *config.Config) *tokens.Manager {
	return tokens.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
}

func newStorageClient(cfg *config.Config) (*storage.Client, error) {
	return storage.NewClient(cfg.Storage)
}

func newOCRAdapter(cfg *config.Config) ocr.Adapter {
	return ocr.NewHTTPAdapter(cfg.OCR)
}
