package accrual

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eorthus/gophermart/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient(server.URL)
	return client, server
}

func TestGetOrderAccrual(t *testing.T) {
	tests := []struct {
		name           string
		orderNumber    string
		handler        func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		expectedResult *models.AccrualResponse
		checkError     func(*testing.T, error)
	}{
		{
			name:        "empty order number",
			orderNumber: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("Server shouldn't be called for empty order number")
			},
			expectError: true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "order number cannot be empty")
			},
		},
		{
			name:        "successful response",
			orderNumber: "4561261212345467",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/orders/4561261212345467", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(models.AccrualResponse{
					Order:   "4561261212345467",
					Status:  models.StatusProcessed,
					Accrual: 500,
				})
			},
			expectedResult: &models.AccrualResponse{
				Order:   "4561261212345467",
				Status:  models.StatusProcessed,
				Accrual: 500,
			},
		},
		{
			name:        "rate limit exceeded",
			orderNumber: "4561261212345467",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
			},
			expectError: true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "unexpected status code 429")
			},
		},
		{
			name:        "internal server error",
			orderNumber: "4561261212345467",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError: true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "unexpected status code 500")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := setupTestServer(t, tt.handler)
			defer server.Close()

			result, err := client.GetOrderAccrual(tt.orderNumber)

			if tt.expectError {
				assert.Error(t, err)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestClient_RateLimitHandling(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}

	client, server := setupTestServer(t, handler)
	defer server.Close()

	_, err := client.GetOrderAccrual("4561261212345467")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 429")
}

// Отдельный тест для OrderNotFoundError
func TestOrderNotFoundError(t *testing.T) {
	err := &OrderNotFoundError{
		OrderNumber: "12345",
	}
	assert.Equal(t, "order 12345 not found", err.Error())
}
