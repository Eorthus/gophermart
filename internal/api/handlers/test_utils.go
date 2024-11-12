package handlers

import (
	"testing"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/service"
	mock_storage "github.com/Eorthus/gophermart/internal/storage/mock"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func setupRouter(t *testing.T) (*chi.Mux, *mock_storage.MockStorage) {
	ctrl := gomock.NewController(t)
	mockStorage := mock_storage.NewMockStorage(ctrl)

	logger := zaptest.NewLogger(t)

	// Initialize the services
	accrualClient := accrual.NewClient("http://test")
	processor := service.NewOrderProcessor(mockStorage, accrualClient, logger)

	// Initialize services with the mock storage
	userService := service.NewUserService(mockStorage)
	orderService := service.NewOrderService(mockStorage, accrualClient, *logger, processor)
	balanceService := service.NewBalanceService(mockStorage)

	// Create router
	r := chi.NewRouter()

	// Auth handlers
	authHandler := NewAuthHandler(userService, logger)
	r.Post("/api/user/register", authHandler.HandleRegister)
	r.Post("/api/user/login", authHandler.HandleLogin)

	// Protected routes
	r.Group(func(r chi.Router) {
		orderHandler := NewOrderHandler(orderService, logger)
		r.Post("/api/user/orders", orderHandler.HandleSubmitOrder)
		r.Get("/api/user/orders", orderHandler.HandleGetOrders)

		balanceHandler := NewBalanceHandler(balanceService, logger)
		r.Get("/api/user/balance", balanceHandler.HandleGetBalance)
		r.Post("/api/user/balance/withdraw", balanceHandler.HandleWithdraw)
		r.Get("/api/user/withdrawals", balanceHandler.HandleGetWithdrawals)
	})

	return r, mockStorage
}
