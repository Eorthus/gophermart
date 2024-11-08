package storage

import (
	"context"
	"errors"

	"github.com/Eorthus/gophermart/internal/models"
)

var (
	ErrUserExists        = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrOrderExists       = errors.New("order already exists")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Storage определяет интерфейс для хранения данных
type Storage interface {
	// User operations
	CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)

	// Order operations
	SaveOrder(ctx context.Context, userID int64, number string) error
	GetOrder(ctx context.Context, number string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status models.OrderStatus, accrual float64) error
	GetOrdersForProcessing(ctx context.Context) ([]models.Order, error)

	// Balance operations
	GetBalance(ctx context.Context, userID int64) (*models.Balance, error)
	UpdateBalance(ctx context.Context, userID int64, delta float64) error

	// Withdrawal operations
	CreateWithdrawal(ctx context.Context, userID int64, orderNumber string, sum float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error)

	// Service operations
	Ping(ctx context.Context) error
	Close() error
}
