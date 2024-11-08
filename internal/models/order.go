package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
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
