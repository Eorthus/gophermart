package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type OrderStatus string

const (
	StatusRegistered OrderStatus = "NEW"        // заказ зарегистрирован, но вознаграждение не рассчитано
	StatusProcessing OrderStatus = "PROCESSING" // расчёт начисления в процессе
	StatusInvalid    OrderStatus = "INVALID"    // заказ не принят к расчёту
	StatusProcessed  OrderStatus = "PROCESSED"  // расчёт начисления окончен
)

type Order struct {
	ID         int64           `json:"-" db:"id"`
	Number     string          `json:"number" db:"number"`
	UserID     int64           `json:"-" db:"user_id"`
	Status     OrderStatus     `json:"status" db:"status"`
	Accrual    sql.NullFloat64 `json:"accrual,omitempty" db:"accrual"`
	UploadedAt time.Time       `json:"uploaded_at" db:"uploaded_at"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	tmp := struct {
		Alias
		Accrual float64 `json:"accrual,omitempty"`
	}{
		Alias: Alias(o),
	}

	if o.Accrual.Valid {
		tmp.Accrual = o.Accrual.Float64
	}

	return json.Marshal(tmp)
}
