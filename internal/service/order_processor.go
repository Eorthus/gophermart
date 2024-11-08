package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
	"go.uber.org/zap"
)

type OrderProcessor struct {
	store         storage.Storage
	accrualClient *accrual.Client
	logger        *zap.Logger
	processingMap sync.Map
	done          chan struct{}
}

func NewOrderProcessor(store storage.Storage, accrualClient *accrual.Client, logger *zap.Logger) *OrderProcessor {
	return &OrderProcessor{
		store:         store,
		accrualClient: accrualClient,
		logger:        logger,
		done:          make(chan struct{}),
	}
}

// Start запускает обработку заказов
func (p *OrderProcessor) Start(ctx context.Context) {
	go p.processOrders(ctx)
}

// Stop останавливает обработку заказов
func (p *OrderProcessor) Stop() {
	close(p.done)
}

// internal/service/order_processor.go
func (p *OrderProcessor) processOrders(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	retryInterval := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case <-ticker.C:
			err := p.processNextBatch(ctx)
			if err != nil {
				// Если получили 404, значит система начисления недоступна
				// Увеличиваем интервал проверки
				if strings.Contains(err.Error(), "404") {
					retryInterval = time.Second * 10 // увеличиваем интервал до 10 секунд
					ticker.Reset(retryInterval)
					continue
				}

				// Для других ошибок также увеличиваем интервал
				if rateLimitErr, ok := err.(*accrual.RateLimitError); ok {
					retryInterval = rateLimitErr.RetryAfter
					ticker.Reset(retryInterval)
				} else {
					// Для остальных ошибок тоже увеличиваем интервал
					retryInterval = time.Second * 5
					ticker.Reset(retryInterval)
				}

				p.logger.Warn("Order processing temporary unavailable",
					zap.Error(err),
					zap.Duration("retry_after", retryInterval))
			} else {
				// При успешной обработке возвращаем нормальный интервал
				if retryInterval != time.Second {
					retryInterval = time.Second
					ticker.Reset(retryInterval)
				}
			}
		}
	}
}

func (p *OrderProcessor) processNextBatch(ctx context.Context) error {
	orders, err := p.store.GetOrdersForProcessing(ctx)
	if err != nil {
		return fmt.Errorf("failed to get orders: %w", err)
	}

	// Если нет заказов для обработки, просто выходим
	if len(orders) == 0 {
		return nil
	}

	for _, order := range orders {
		// Пропускаем заказы с финальными статусами
		if order.Status == models.StatusProcessed ||
			order.Status == models.StatusInvalid {
			continue
		}

		// Проверяем, не обрабатывается ли уже этот заказ
		if _, exists := p.processingMap.LoadOrStore(order.Number, true); exists {
			continue
		}

		// Делаем запрос к системе начисления
		accrual, err := p.accrualClient.GetOrderAccrual(order.Number)
		p.processingMap.Delete(order.Number)

		if err != nil {
			// Если система начисления недоступна, просто пропускаем заказ
			// Он будет обработан при следующей попытке
			if strings.Contains(err.Error(), "404") {
				continue
			}
			return err
		}

		if accrual == nil {
			continue
		}

		// Обновляем статус заказа
		err = p.store.UpdateOrderStatus(ctx, order.Number, accrual.Status, accrual.Accrual)
		if err != nil {
			p.logger.Error("Failed to update order status",
				zap.String("order", order.Number),
				zap.Error(err))
		}
	}

	return nil
}
