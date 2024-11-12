package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
	"github.com/Eorthus/gophermart/internal/utils"
	"go.uber.org/zap"
)

type OrderService struct {
	store          storage.Storage
	logger         zap.Logger
	accrualClient  *accrual.Client
	orderProcessor *OrderProcessor
}

func NewOrderService(store storage.Storage, accrualClient *accrual.Client, logger zap.Logger, orderProcessor *OrderProcessor) *OrderService {
	return &OrderService{
		store:          store,
		accrualClient:  accrualClient,
		logger:         logger,
		orderProcessor: orderProcessor,
	}
}

func (s *OrderService) SubmitOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Валидируем номер заказа по алгоритму Луна
	if !utils.ValidateLuhn(orderNumber) {
		s.logger.Info("Invalid order number format by Luhn algorithm",
			zap.String("order_number", orderNumber))
		return apperrors.ErrInvalidOrder
	}

	err := s.store.SaveOrder(ctx, userID, orderNumber)
	if err != nil {
		if errors.Is(err, apperrors.ErrOrderExistsForUser) {
			return apperrors.ErrOrderExistsForUser
		}
		if errors.Is(err, apperrors.ErrOrderExistsForOther) {
			return apperrors.ErrOrderExistsForOther
		}
		return fmt.Errorf("failed to save order: %w", err)
	}

	// Добавляем заказ в очередь на обработку
	s.orderProcessor.AddOrder(orderNumber)

	return nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.store.GetUserOrders(ctx, userID)
}
