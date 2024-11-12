package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/models"
	mock_storage "github.com/Eorthus/gophermart/internal/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Тестируем только добавление заказа в очередь
func TestOrderProcessor_AddOrder(t *testing.T) {
	logger := zap.NewExample()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	accrualClient := accrual.NewClient(server.URL)
	processor := NewOrderProcessor(mockStorage, accrualClient, logger)

	orderNumber := "4561261212345467"
	processor.AddOrder(orderNumber)

	select {
	case receivedOrder := <-processor.ordersChan:
		assert.Equal(t, orderNumber, receivedOrder)
	case <-time.After(100 * time.Millisecond):
		t.Error("Order was not added to the queue")
	}
}

// Тестируем только корректную остановку процессора
func TestOrderProcessor_Stop(t *testing.T) {
	logger := zap.NewExample()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	accrualClient := accrual.NewClient(server.URL)
	processor := NewOrderProcessor(mockStorage, accrualClient, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processor.Start(ctx)
	processor.Stop()

	select {
	case <-processor.done:
		// Успешно - канал закрыт
	case <-time.After(100 * time.Millisecond):
		t.Error("Processor was not stopped properly")
	}
}

// Простой тест проверки заказа
func TestOrderProcessor_CheckOrder(t *testing.T) {
	logger := zap.NewExample()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)

	// Настраиваем тестовый сервер с простым ответом
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.AccrualResponse{
			Order:   "4561261212345467",
			Status:  models.StatusProcessed,
			Accrual: 500,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	accrualClient := accrual.NewClient(server.URL)

	// Настраиваем простые ожидания для хранилища
	mockStorage.EXPECT().
		GetOrder(gomock.Any(), "4561261212345467").
		Return(&models.Order{
			ID:     1,
			Number: "4561261212345467",
			UserID: 1,
			Status: models.StatusRegistered,
		}, nil)

	mockStorage.EXPECT().
		UpdateOrderStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	mockStorage.EXPECT().
		UpdateBalance(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	processor := NewOrderProcessor(mockStorage, accrualClient, logger)
	err := processor.checkOrder(context.Background(), "4561261212345467")
	assert.NoError(t, err)
}
