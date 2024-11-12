package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	mock_storage "github.com/Eorthus/gophermart/internal/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestOrderService_SubmitOrder(t *testing.T) {
	logger, _ := zap.NewProduction()

	tests := []struct {
		name           string
		userID         int64
		orderNumber    string
		mockSetup      func(*mock_storage.MockStorage)
		expectedError  error
		processorSetup func(*OrderProcessor)
	}{
		{
			name:        "valid new order",
			userID:      1,
			orderNumber: "4561261212345467", // валидный номер по алгоритму Луна
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(nil)
			},
			processorSetup: func(op *OrderProcessor) {
				// Процессор должен получить заказ для обработки
			},
			expectedError: nil,
		},
		{
			name:        "invalid order number",
			userID:      1,
			orderNumber: "12345", // невалидный номер
			mockSetup: func(ms *mock_storage.MockStorage) {
				// Хранилище не должно вызываться для невалидного номера
			},
			processorSetup: func(op *OrderProcessor) {
				// Процессор не должен получать невалидный заказ
			},
			expectedError: apperrors.ErrInvalidOrder,
		},
		{
			name:        "order exists for same user",
			userID:      1,
			orderNumber: "4561261212345467",
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(apperrors.ErrOrderExistsForUser)
			},
			processorSetup: func(op *OrderProcessor) {
				// Процессор не должен получать существующий заказ
			},
			expectedError: apperrors.ErrOrderExistsForUser,
		},
		{
			name:        "order exists for other user",
			userID:      1,
			orderNumber: "4561261212345467",
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(apperrors.ErrOrderExistsForOther)
			},
			processorSetup: func(op *OrderProcessor) {
				// Процессор не должен получать заказ другого пользователя
			},
			expectedError: apperrors.ErrOrderExistsForOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mock_storage.NewMockStorage(ctrl)
			tt.mockSetup(mockStorage)

			accrualClient := accrual.NewClient("http://test")
			processor := NewOrderProcessor(mockStorage, accrualClient, logger)
			tt.processorSetup(processor)

			service := NewOrderService(mockStorage, accrualClient, *logger, processor)
			err := service.SubmitOrder(context.Background(), tt.userID, tt.orderNumber)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderService_GetUserOrders(t *testing.T) {
	logger, _ := zap.NewProduction()

	tests := []struct {
		name           string
		userID         int64
		mockSetup      func(*mock_storage.MockStorage)
		expectedOrders []models.Order
		expectedError  error
	}{
		{
			name:   "user has orders",
			userID: 1,
			mockSetup: func(ms *mock_storage.MockStorage) {
				orders := []models.Order{
					{
						ID:      1,
						Number:  "4561261212345467",
						UserID:  1,
						Status:  models.StatusProcessed,
						Accrual: sql.NullFloat64{Float64: 500, Valid: true},
					},
					{
						ID:      2,
						Number:  "4561261212345468",
						UserID:  1,
						Status:  models.StatusProcessing,
						Accrual: sql.NullFloat64{Valid: false},
					},
				}
				ms.EXPECT().
					GetUserOrders(gomock.Any(), int64(1)).
					Return(orders, nil)
			},
			expectedOrders: []models.Order{
				{
					ID:      1,
					Number:  "4561261212345467",
					UserID:  1,
					Status:  models.StatusProcessed,
					Accrual: sql.NullFloat64{Float64: 500, Valid: true},
				},
				{
					ID:      2,
					Number:  "4561261212345468",
					UserID:  1,
					Status:  models.StatusProcessing,
					Accrual: sql.NullFloat64{Valid: false},
				},
			},
			expectedError: nil,
		},
		{
			name:   "user has no orders",
			userID: 2,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetUserOrders(gomock.Any(), int64(2)).
					Return([]models.Order{}, nil)
			},
			expectedOrders: []models.Order{},
			expectedError:  nil,
		},
		{
			name:   "storage error",
			userID: 3,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetUserOrders(gomock.Any(), int64(3)).
					Return(nil, errors.New("storage error"))
			},
			expectedOrders: nil,
			expectedError:  errors.New("storage error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mock_storage.NewMockStorage(ctrl)
			tt.mockSetup(mockStorage)

			accrualClient := accrual.NewClient("http://test")
			processor := NewOrderProcessor(mockStorage, accrualClient, logger)
			service := NewOrderService(mockStorage, accrualClient, *logger, processor)

			orders, err := service.GetUserOrders(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, orders)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOrders, orders)
			}
		})
	}
}
