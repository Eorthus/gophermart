package service

import (
	"context"
	"errors"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
	"github.com/Eorthus/gophermart/internal/utils"
)

type BalanceService struct {
	store storage.Storage
}

func NewBalanceService(store storage.Storage) *BalanceService {
	return &BalanceService{store: store}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	return s.store.GetBalance(ctx, userID)
}

func (s *BalanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	// Проверяем валидность номера заказа
	if !utils.ValidateLuhn(orderNumber) {
		return apperrors.ErrInvalidOrder
	}

	// Проверяем сумму списания
	if amount <= 0 {
		return apperrors.ErrInvalidWithdraw
	}

	// Создаём запись о списании
	err := s.store.CreateWithdrawal(ctx, userID, orderNumber, amount)
	if err != nil {
		if errors.Is(err, storage.ErrInsufficientFunds) {
			return apperrors.ErrInsufficientFunds
		}
		return err
	}

	return nil
}

func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	return s.store.GetWithdrawals(ctx, userID)
}
