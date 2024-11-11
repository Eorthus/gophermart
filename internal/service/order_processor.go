package service

import (
	"context"
	"fmt"
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
	ordersChan    chan string // Канал для новых заказов
	done          chan struct{}
}

func NewOrderProcessor(store storage.Storage, accrualClient *accrual.Client, logger *zap.Logger) *OrderProcessor {
	return &OrderProcessor{
		store:         store,
		accrualClient: accrualClient,
		logger:        logger,
		ordersChan:    make(chan string, 100), // Буфер для новых заказов
		done:          make(chan struct{}),
	}
}

// AddOrder добавляет заказ в очередь на обработку
func (p *OrderProcessor) AddOrder(orderNumber string) {
	select {
	case p.ordersChan <- orderNumber:
		p.logger.Debug("Order added to processing queue", zap.String("order", orderNumber))
	default:
		p.logger.Warn("Processing queue is full", zap.String("order", orderNumber))
	}
}

func (p *OrderProcessor) Start(ctx context.Context) {
	go p.processOrders(ctx)
}

func (p *OrderProcessor) Stop() {
	close(p.done)
}

func (p *OrderProcessor) processOrders(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	pending := make(map[string]time.Time) // Карта заказов в обработке

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case orderNumber := <-p.ordersChan:
			// Добавляем новый заказ в список ожидающих
			pending[orderNumber] = time.Now()
		case <-ticker.C:
			if len(pending) == 0 {
				continue
			}

			// Проверяем статус только тех заказов, которые в обработке
			for orderNumber, lastCheck := range pending {
				// Пропускаем заказы, которые проверялись менее 5 секунд назад
				if time.Since(lastCheck) < 5*time.Second {
					continue
				}

				if err := p.checkOrder(ctx, orderNumber); err != nil {
					if _, ok := err.(*accrual.RateLimitError); ok {
						// Обрабатываем rate limit
						time.Sleep(time.Second)
						continue
					}
					p.logger.Error("Failed to check order",
						zap.String("order", orderNumber),
						zap.Error(err))
					continue
				}

				// Проверяем статус заказа
				order, err := p.store.GetOrder(ctx, orderNumber)
				if err != nil {
					p.logger.Error("Failed to get order status",
						zap.String("order", orderNumber),
						zap.Error(err))
					continue
				}

				// Если заказ обработан или отклонен, удаляем его из очереди
				if order.Status == models.StatusProcessed ||
					order.Status == models.StatusInvalid {
					delete(pending, orderNumber)
				} else {
					// Обновляем время последней проверки
					pending[orderNumber] = time.Now()
				}
			}
		}
	}
}

func (p *OrderProcessor) checkOrder(ctx context.Context, orderNumber string) error {
	accrual, err := p.accrualClient.GetOrderAccrual(orderNumber)
	if err != nil {
		return err
	}

	if accrual == nil {
		return nil
	}

	err = p.store.UpdateOrderStatus(ctx, orderNumber, accrual.Status, accrual.Accrual)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if accrual.Status == models.StatusProcessed && accrual.Accrual > 0 {
		// Получаем заказ для получения ID пользователя
		order, err := p.store.GetOrder(ctx, orderNumber)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		if err := p.store.UpdateBalance(ctx, order.UserID, accrual.Accrual); err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}
	}

	return nil
}
