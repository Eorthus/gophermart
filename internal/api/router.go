// internal/api/router.go
package api

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/Eorthus/gophermart/internal/api/handlers"
	"github.com/Eorthus/gophermart/internal/config"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/service"
)

func NewRouter(
	cfg *config.Config,
	userService *service.UserService,
	orderService *service.OrderService,
	balanceService *service.BalanceService,
	logger *zap.Logger,
) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger(logger)) // Используем существующий Logger
	r.Use(middleware.GzipMiddleware)

	// Auth handlers
	authHandler := handlers.NewAuthHandler(userService, logger)
	r.Post("/api/user/register", authHandler.HandleRegister)
	r.Post("/api/user/login", authHandler.HandleLogin)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(logger))

		// Order handlers
		orderHandler := handlers.NewOrderHandler(orderService, logger)
		r.Post("/api/user/orders", orderHandler.HandleSubmitOrder)
		r.Get("/api/user/orders", orderHandler.HandleGetOrders)

		// Balance handlers
		balanceHandler := handlers.NewBalanceHandler(balanceService, logger)
		r.Get("/api/user/balance", balanceHandler.HandleGetBalance)
		r.Post("/api/user/balance/withdraw", balanceHandler.HandleWithdraw)
		r.Get("/api/user/withdrawals", balanceHandler.HandleGetWithdrawals)
	})

	return r
}
