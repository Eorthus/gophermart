package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleGetBalance(t *testing.T) {
	r, mockStorage := setupRouter(t)

	tests := []struct {
		name           string
		setupMocks     func()
		expectedStatus int
		expectedBody   *models.Balance
	}{
		{
			name: "Successful balance retrieval",
			setupMocks: func() {
				mockStorage.EXPECT().
					GetBalance(gomock.Any(), int64(1)).
					Return(&models.Balance{
						Current:   100.50,
						Withdrawn: 50.25,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &models.Balance{
				Current:   100.50,
				Withdrawn: 50.25,
			},
		},
		{
			name: "Empty balance",
			setupMocks: func() {
				mockStorage.EXPECT().
					GetBalance(gomock.Any(), int64(1)).
					Return(&models.Balance{
						Current:   0,
						Withdrawn: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &models.Balance{
				Current:   0,
				Withdrawn: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			req := httptest.NewRequest("GET", "/api/user/balance", nil)
			req = addAuthCookie(req, "1")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedBody != nil {
				var response models.Balance
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, &response)
			}
		})
	}
}

func TestHandleWithdraw(t *testing.T) {
	r, mockStorage := setupRouter(t)

	tests := []struct {
		name           string
		request        models.WithdrawRequest
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Successful withdrawal",
			request: models.WithdrawRequest{
				Order: "4561261212345467",
				Sum:   100.0,
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					CreateWithdrawal(gomock.Any(), int64(1), "4561261212345467", float64(100.0)).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid order number",
			request: models.WithdrawRequest{
				Order: "12345",
				Sum:   100.0,
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Insufficient funds",
			request: models.WithdrawRequest{
				Order: "4561261212345467",
				Sum:   1000.0,
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					CreateWithdrawal(gomock.Any(), int64(1), "4561261212345467", float64(1000.0)).
					Return(apperrors.ErrInsufficientFunds)
			},
			expectedStatus: http.StatusPaymentRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/user/balance/withdraw", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = addAuthCookie(req, "1")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHandleGetWithdrawals(t *testing.T) {
	r, mockStorage := setupRouter(t)

	withdrawalTime := time.Now()

	tests := []struct {
		name              string
		setupMocks        func()
		expectedStatus    int
		expectedWithdraws []models.Withdrawal
	}{
		{
			name: "User has withdrawals",
			setupMocks: func() {
				mockStorage.EXPECT().
					GetWithdrawals(gomock.Any(), int64(1)).
					Return([]models.Withdrawal{
						{
							OrderNumber: "4561261212345467",
							Sum:         100.0,
							ProcessedAt: withdrawalTime,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedWithdraws: []models.Withdrawal{
				{
					OrderNumber: "4561261212345467",
					Sum:         100.0,
					ProcessedAt: withdrawalTime,
				},
			},
		},
		{
			name: "User has no withdrawals",
			setupMocks: func() {
				mockStorage.EXPECT().
					GetWithdrawals(gomock.Any(), int64(1)).
					Return([]models.Withdrawal{}, nil)
			},
			expectedStatus:    http.StatusNoContent,
			expectedWithdraws: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			req := httptest.NewRequest("GET", "/api/user/withdrawals", nil)
			req = addAuthCookie(req, "1")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedWithdraws != nil {
				var response []models.Withdrawal
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				assert.Equal(t, len(tt.expectedWithdraws), len(response))
				for i := range response {
					assert.Equal(t, tt.expectedWithdraws[i].OrderNumber, response[i].OrderNumber)
					assert.Equal(t, tt.expectedWithdraws[i].Sum, response[i].Sum)
				}
			}
		})
	}
}
