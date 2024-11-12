package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addAuthCookie(req *http.Request, userID string) *http.Request {
	w := httptest.NewRecorder()
	middleware.SetAuthCookie(w, userID)
	resp := w.Result()
	defer resp.Body.Close()
	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func TestHandleSubmitOrder(t *testing.T) {
	r, mockStorage := setupRouter(t)

	tests := []struct {
		name           string
		orderNumber    string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:        "Valid order number",
			orderNumber: "4561261212345467",
			setupMocks: func() {
				// Мок для SaveOrder
				mockStorage.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(nil)

				// Дополнительные моки для OrderProcessor
				mockStorage.EXPECT().
					GetOrder(gomock.Any(), "4561261212345467").
					AnyTimes().
					Return(&models.Order{
						ID:     1,
						Number: "4561261212345467",
						UserID: 1,
						Status: models.StatusRegistered,
					}, nil)
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "Invalid order number",
			orderNumber:    "12345",
			setupMocks:     func() {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "Order exists for current user",
			orderNumber: "4561261212345467",
			setupMocks: func() {
				mockStorage.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(apperrors.ErrOrderExistsForUser)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Order exists for another user",
			orderNumber: "4561261212345467",
			setupMocks: func() {
				mockStorage.EXPECT().
					SaveOrder(gomock.Any(), int64(1), "4561261212345467").
					Return(apperrors.ErrOrderExistsForOther)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			req := httptest.NewRequest("POST", "/api/user/orders", bytes.NewBufferString(tt.orderNumber))
			req.Header.Set("Content-Type", "text/plain")
			req = addAuthCookie(req, "1")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHandleGetOrders(t *testing.T) {
	r, mockStorage := setupRouter(t)

	tests := []struct {
		name           string
		setupMocks     func()
		expectedStatus int
		expectedOrders []models.Order
	}{
		{
			name: "User has orders",
			setupMocks: func() {
				orders := []models.Order{
					{
						Number:     "4561261212345467",
						Status:     models.StatusProcessed,
						UploadedAt: time.Now(),
					},
				}
				mockStorage.EXPECT().
					GetUserOrders(gomock.Any(), int64(1)).
					Return(orders, nil)
			},
			expectedStatus: http.StatusOK,
			expectedOrders: []models.Order{
				{
					Number: "4561261212345467",
					Status: models.StatusProcessed,
				},
			},
		},
		{
			name: "User has no orders",
			setupMocks: func() {
				mockStorage.EXPECT().
					GetUserOrders(gomock.Any(), int64(1)).
					Return([]models.Order{}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedOrders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			req := httptest.NewRequest("GET", "/api/user/orders", nil)
			req = addAuthCookie(req, "1")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedOrders != nil {
				var response []models.Order
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				for i := range response {
					assert.Equal(t, tt.expectedOrders[i].Number, response[i].Number)
					assert.Equal(t, tt.expectedOrders[i].Status, response[i].Status)
				}
			}
		})
	}
}
