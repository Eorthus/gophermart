package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	_ "github.com/lib/pq"
)

type DatabaseStorage struct {
	db *sql.DB
}

func NewDatabaseStorage(ctx context.Context, dsn string) (*DatabaseStorage, error) {
	// Сначала подключаемся к postgres для создания БД
	baseDB, err := sql.Open("postgres", getDefaultDSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer baseDB.Close()

	// Создаем базу данных, если её нет
	dbName := getDatabaseName(dsn)
	_, err = baseDB.ExecContext(ctx, fmt.Sprintf(`
        CREATE DATABASE %s;
    `, dbName))
	if err != nil {
		// Игнорируем ошибку, если база уже существует
		log.Printf("Database created")
	}

	// Теперь подключаемся к созданной базе данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &DatabaseStorage{db: db}
	if err := storage.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return storage, nil
}

// getDefaultDSN возвращает DSN для подключения к postgres
func getDefaultDSN(dsn string) string {
	// Заменяем имя базы данных на postgres в строке подключения
	return strings.Replace(dsn, getDatabaseName(dsn), "postgres", 1)
}

// getDatabaseName извлекает имя базы данных из DSN
func getDatabaseName(dsn string) string {
	// Простой парсер DSN - в реальном проекте лучше использовать более надежный способ
	parts := strings.Split(dsn, "/")
	if len(parts) < 2 {
		return "gophermart" // значение по умолчанию
	}
	dbName := strings.Split(parts[len(parts)-1], "?")[0]
	return dbName
}

func (s *DatabaseStorage) initSchema(ctx context.Context) error {
	query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        login VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS orders (
        id SERIAL PRIMARY KEY,
        number VARCHAR(255) UNIQUE NOT NULL,
        user_id INTEGER REFERENCES users(id),
        status VARCHAR(50) NOT NULL,
        accrual DECIMAL(10, 2),
        uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS balances (
        user_id INTEGER PRIMARY KEY REFERENCES users(id),
        current DECIMAL(10, 2) NOT NULL DEFAULT 0,
        withdrawn DECIMAL(10, 2) NOT NULL DEFAULT 0
    );

    CREATE TABLE IF NOT EXISTS withdrawals (
        id SERIAL PRIMARY KEY,
        user_id INTEGER REFERENCES users(id),
        order_number VARCHAR(255) NOT NULL,
        sum DECIMAL(10, 2) NOT NULL,
        processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// User operations implementation
func (s *DatabaseStorage) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	query := `
        INSERT INTO users (login, password_hash) 
        VALUES ($1, $2) 
        RETURNING id, login, password_hash, created_at`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, login, passwordHash).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt,
	)
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}

	// Create initial balance record
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO balances (user_id, current, withdrawn) VALUES ($1, 0, 0)",
		user.ID,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *DatabaseStorage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
        SELECT id, login, password_hash, created_at 
        FROM users 
        WHERE login = $1`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// Order operations implementation
func (s *DatabaseStorage) SaveOrder(ctx context.Context, userID int64, number string) error {
	// Сначала проверяем, существует ли заказ
	var existingUserID sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		"SELECT user_id FROM orders WHERE number = $1",
		number,
	).Scan(&existingUserID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing order: %w", err)
	}

	// Если заказ существует
	if err != sql.ErrNoRows {
		// Если заказ принадлежит текущему пользователю
		if existingUserID.Valid && existingUserID.Int64 == userID {
			return apperrors.ErrOrderExistsForUser
		}
		// Если заказ принадлежит другому пользователю
		return apperrors.ErrOrderExistsForOther
	}

	// Сохраняем новый заказ
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO orders (number, user_id, status) VALUES ($1, $2, $3)",
		number, userID, models.StatusRegistered,
	)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	return nil
}

func (s *DatabaseStorage) GetOrder(ctx context.Context, number string) (*models.Order, error) {
	query := `
        SELECT id, number, user_id, status, accrual, uploaded_at 
        FROM orders 
        WHERE number = $1`

	var order models.Order
	err := s.db.QueryRowContext(ctx, query, number).Scan(
		&order.ID,
		&order.Number,
		&order.UserID,
		&order.Status,
		&order.Accrual, // теперь это sql.NullFloat64
		&order.UploadedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *DatabaseStorage) GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	query := `
        SELECT id, number, user_id, status, accrual, uploaded_at 
        FROM orders 
        WHERE user_id = $1 
        ORDER BY uploaded_at DESC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.Number, &order.UserID,
			&order.Status, &order.Accrual, &order.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *DatabaseStorage) UpdateOrderStatus(ctx context.Context, number string, status models.OrderStatus, accrual float64) error {
	query := `
        UPDATE orders 
        SET status = $1, accrual = $2 
        WHERE number = $3`

	// Если accrual равен 0, сохраняем NULL
	var accrualValue interface{}
	if accrual > 0 {
		accrualValue = accrual
	} else {
		accrualValue = nil
	}

	result, err := s.db.ExecContext(ctx, query, status, accrualValue, number)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Balance operations implementation
func (s *DatabaseStorage) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	query := `
        SELECT current, withdrawn 
        FROM balances 
        WHERE user_id = $1`

	balance := &models.Balance{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Balance{}, nil
		}
		return nil, err
	}
	return balance, nil
}

func (s *DatabaseStorage) UpdateBalance(ctx context.Context, userID int64, delta float64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current float64
	err = tx.QueryRowContext(ctx,
		"SELECT current FROM balances WHERE user_id = $1 FOR UPDATE",
		userID,
	).Scan(&current)
	if err != nil {
		return err
	}

	// Проверяем, достаточно ли средств при списании
	if delta < 0 && current+delta < 0 {
		return ErrInsufficientFunds
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE balances SET current = current + $1 WHERE user_id = $2",
		delta, userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Withdrawal operations implementation
func (s *DatabaseStorage) CreateWithdrawal(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Проверяем баланс и блокируем запись
	var current float64
	err = tx.QueryRowContext(ctx,
		"SELECT current FROM balances WHERE user_id = $1 FOR UPDATE",
		userID,
	).Scan(&current)
	if err != nil {
		return err
	}

	if current < sum {
		return ErrInsufficientFunds
	}

	// Создаем запись о списании
	_, err = tx.ExecContext(ctx,
		"INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)",
		userID, orderNumber, sum,
	)
	if err != nil {
		return err
	}

	// Обновляем баланс
	_, err = tx.ExecContext(ctx,
		"UPDATE balances SET current = current - $1, withdrawn = withdrawn + $1 WHERE user_id = $2",
		sum, userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *DatabaseStorage) GetWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	query := `
        SELECT id, user_id, order_number, sum, processed_at 
        FROM withdrawals 
        WHERE user_id = $1 
        ORDER BY processed_at DESC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, rows.Err()
}

func (s *DatabaseStorage) GetOrdersForProcessing(ctx context.Context) ([]models.Order, error) {
	query := `
        SELECT id, number, user_id, status, accrual, uploaded_at 
        FROM orders 
        WHERE status NOT IN ('PROCESSED', 'INVALID') 
        ORDER BY uploaded_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.Number,
			&order.UserID,
			&order.Status,
			&order.Accrual, // теперь это sql.NullFloat64
			&order.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// Service operations implementation
func (s *DatabaseStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *DatabaseStorage) Close() error {
	return s.db.Close()
}

// Utility functions
func isPgUniqueViolation(err error) bool {
	// Implement PostgreSQL unique constraint violation check
	return false // TODO: implement
}
