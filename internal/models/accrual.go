package models

// AccrualResponse представляет ответ от системы начисления баллов
type AccrualResponse struct {
	Order   string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual,omitempty"`
}
