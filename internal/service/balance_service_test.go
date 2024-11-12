package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	mock_storage "github.com/Eorthus/gophermart/internal/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBalanceService_GetBalance(t *testing.T) {
	tests := []struct {
		name            string
		userID          int64
		mockSetup       func(*mock_storage.MockStorage)
		expectedBalance *models.Balance
		expectedError   error
	}{
		{
			name:   "successful balance retrieval",
			userID: 1,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetBalance(gomock.Any(), int64(1)).
					Return(&models.Balance{Current: 100.0, Withdrawn: 50.0}, nil)
			},
			expectedBalance: &models.Balance{Current: 100.0, Withdrawn: 50.0},
			expectedError:   nil,
		},
		{
			name:   "zero balance",
			userID: 2,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetBalance(gomock.Any(), int64(2)).
					Return(&models.Balance{Current: 0, Withdrawn: 0}, nil)
			},
			expectedBalance: &models.Balance{Current: 0, Withdrawn: 0},
			expectedError:   nil,
		},
		{
			name:   "storage error",
			userID: 3,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetBalance(gomock.Any(), int64(3)).
					Return(nil, errors.New("storage error"))
			},
			expectedBalance: nil,
			expectedError:   errors.New("storage error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mock_storage.NewMockStorage(ctrl)
			tt.mockSetup(mockStorage)

			service := NewBalanceService(mockStorage)
			balance, err := service.GetBalance(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, balance)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBalance, balance)
			}
		})
	}
}

func TestBalanceService_Withdraw(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		orderNumber   string
		amount        float64
		mockSetup     func(*mock_storage.MockStorage)
		expectedError error
	}{
		{
			name:        "successful withdrawal",
			userID:      1,
			orderNumber: "4561261212345467",
			amount:      50.0,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					CreateWithdrawal(gomock.Any(), int64(1), "4561261212345467", 50.0).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:          "invalid order number",
			userID:        1,
			orderNumber:   "12345",
			amount:        50.0,
			mockSetup:     func(ms *mock_storage.MockStorage) {},
			expectedError: apperrors.ErrInvalidOrder,
		},
		{
			name:        "insufficient funds",
			userID:      1,
			orderNumber: "4561261212345467",
			amount:      1000.0,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					CreateWithdrawal(gomock.Any(), int64(1), "4561261212345467", 1000.0).
					Return(apperrors.ErrInsufficientFunds)
			},
			expectedError: apperrors.ErrInsufficientFunds,
		},
		{
			name:          "negative amount",
			userID:        1,
			orderNumber:   "4561261212345467",
			amount:        -50.0,
			mockSetup:     func(ms *mock_storage.MockStorage) {},
			expectedError: apperrors.ErrInvalidWithdraw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mock_storage.NewMockStorage(ctrl)
			tt.mockSetup(mockStorage)

			service := NewBalanceService(mockStorage)
			err := service.Withdraw(context.Background(), tt.userID, tt.orderNumber, tt.amount)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	testTime := time.Now()

	tests := []struct {
		name              string
		userID            int64
		mockSetup         func(*mock_storage.MockStorage)
		expectedWithdraws []models.Withdrawal
		expectedError     error
	}{
		{
			name:   "user has withdrawals",
			userID: 1,
			mockSetup: func(ms *mock_storage.MockStorage) {
				withdrawals := []models.Withdrawal{
					{
						ID:          1,
						UserID:      1,
						OrderNumber: "4561261212345467",
						Sum:         50.0,
						ProcessedAt: testTime,
					},
					{
						ID:          2,
						UserID:      1,
						OrderNumber: "4561261212345468",
						Sum:         30.0,
						ProcessedAt: testTime,
					},
				}
				ms.EXPECT().
					GetWithdrawals(gomock.Any(), int64(1)).
					Return(withdrawals, nil)
			},
			expectedWithdraws: []models.Withdrawal{
				{
					ID:          1,
					UserID:      1,
					OrderNumber: "4561261212345467",
					Sum:         50.0,
					ProcessedAt: testTime,
				},
				{
					ID:          2,
					UserID:      1,
					OrderNumber: "4561261212345468",
					Sum:         30.0,
					ProcessedAt: testTime,
				},
			},
			expectedError: nil,
		},
		{
			name:   "user has no withdrawals",
			userID: 2,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetWithdrawals(gomock.Any(), int64(2)).
					Return([]models.Withdrawal{}, nil)
			},
			expectedWithdraws: []models.Withdrawal{},
			expectedError:     nil,
		},
		{
			name:   "storage error",
			userID: 3,
			mockSetup: func(ms *mock_storage.MockStorage) {
				ms.EXPECT().
					GetWithdrawals(gomock.Any(), int64(3)).
					Return(nil, errors.New("storage error"))
			},
			expectedWithdraws: nil,
			expectedError:     errors.New("storage error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mock_storage.NewMockStorage(ctrl)
			tt.mockSetup(mockStorage)

			service := NewBalanceService(mockStorage)
			withdrawals, err := service.GetWithdrawals(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, withdrawals)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWithdraws, withdrawals)
			}
		})
	}
}
