package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type OrderProcessor struct {
	orders     map[string]*AccrualResponse
	mu         sync.RWMutex
	requestLog map[string][]time.Time // для отслеживания частоты запросов
	logMu      sync.Mutex
}

func NewOrderProcessor() *OrderProcessor {
	return &OrderProcessor{
		orders:     make(map[string]*AccrualResponse),
		requestLog: make(map[string][]time.Time),
	}
}

func (p *OrderProcessor) cleanupOldRequests() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		p.logMu.Lock()
		now := time.Now()
		for ip, times := range p.requestLog {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < time.Minute {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(p.requestLog, ip)
			} else {
				p.requestLog[ip] = valid
			}
		}
		p.logMu.Unlock()
	}
}

func (p *OrderProcessor) checkRateLimit(ip string) bool {
	p.logMu.Lock()
	defer p.logMu.Unlock()

	now := time.Now()
	times := p.requestLog[ip]

	// Очищаем старые запросы
	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < time.Minute {
			valid = append(valid, t)
		}
	}

	// Проверяем лимит (20 запросов в минуту)
	if len(valid) >= 20 {
		p.requestLog[ip] = valid
		return false
	}

	// Добавляем новый запрос
	valid = append(valid, now)
	p.requestLog[ip] = valid
	return true
}

func main() {
	processor := NewOrderProcessor()
	go processor.cleanupOldRequests()

	r := chi.NewRouter()

	r.Get("/api/orders/{number}", func(w http.ResponseWriter, r *http.Request) {
		// Проверка rate limit
		ip := r.RemoteAddr
		if !processor.checkRateLimit(ip) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("No more than 20 requests per minute allowed"))
			return
		}

		number := chi.URLParam(r, "number")

		processor.mu.RLock()
		order, exists := processor.orders[number]
		processor.mu.RUnlock()

		if !exists {
			// Имитируем случай, когда заказ еще не зарегистрирован (5% случаев)
			if rand.Float64() < 0.05 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			order = &AccrualResponse{
				Order:  number,
				Status: "REGISTERED",
			}

			processor.mu.Lock()
			processor.orders[number] = order
			processor.mu.Unlock()

			// Запускаем асинхронную обработку
			go func() {
				// REGISTERED -> PROCESSING
				time.Sleep(5 * time.Second)
				processor.mu.Lock()
				order.Status = "PROCESSING"
				processor.mu.Unlock()

				// PROCESSING -> PROCESSED/INVALID
				time.Sleep(5 * time.Second)
				processor.mu.Lock()
				if rand.Float64() < 0.1 { // 10% шанс на INVALID
					order.Status = "INVALID"
				} else {
					order.Status = "PROCESSED"
					order.Accrual = float64(rand.Intn(2000)) / 10 // 0-200 баллов
				}
				processor.mu.Unlock()
			}()
		}

		// Симулируем случайные ошибки сервера (1% случаев)
		if rand.Float64() < 0.01 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		processor.mu.RLock()
		response := *order // Создаем копию для ответа
		processor.mu.RUnlock()

		// Если статус не PROCESSED, убираем accrual из ответа
		if response.Status != "PROCESSED" {
			response.Accrual = 0
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Starting mock accrual system on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
