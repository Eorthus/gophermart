package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
	"go.uber.org/zap"
)

type OrderService struct {
	store         storage.Storage
	logger        zap.Logger
	accrualClient *accrual.Client
}

func NewOrderService(store storage.Storage, accrualClient *accrual.Client, logger zap.Logger) *OrderService {
	return &OrderService{
		store:         store,
		accrualClient: accrualClient,
		logger:        logger,
	}
}

func (s *OrderService) SubmitOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Валидируем номер заказа по алгоритму Луна
	if !validateLuhn(orderNumber) {
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

	return nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.store.GetUserOrders(ctx, userID)
}

func validateLuhn(number string) bool {
	// Проверяем, что строка содержит только цифры
	if !regexp.MustCompile(`^\d+$`).MatchString(number) {
		return false
	}

	sum := 0
	nDigits := len(number)
	parity := nDigits % 2

	for i := 0; i < nDigits; i++ {
		digit := int(number[i] - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	return sum%10 == 0
}
