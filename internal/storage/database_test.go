package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := &DatabaseStorage{db: db}
	ctx := context.Background()

	// Фиксированное время для тестов
	fixedTime := time.Date(2022, time.November, 10, 10, 0, 0, 0, time.UTC)

	t.Run("CreateUser - Success", func(t *testing.T) {
		login := "test_user"
		passwordHash := "hashed_password"
		userID := int64(1)

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(login, passwordHash).
			WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
				AddRow(userID, login, passwordHash, time.Now()))

		mock.ExpectExec("INSERT INTO balances").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		user, err := store.CreateUser(ctx, login, passwordHash)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, login, user.Login)
	})

	t.Run("GetUserByLogin - User Exists", func(t *testing.T) {
		login := "test_user"
		passwordHash := "hashed_password"
		userID := int64(1)

		rows := sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
			AddRow(userID, login, passwordHash, time.Now())
		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users").
			WithArgs(login).
			WillReturnRows(rows)

		user, err := store.GetUserByLogin(ctx, login)
		assert.NoError(t, err)
		assert.Equal(t, login, user.Login)
		assert.Equal(t, passwordHash, user.PasswordHash)
	})

	t.Run("GetUserByLogin - User Not Found", func(t *testing.T) {
		login := "non_existent_user"

		mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users").
			WithArgs(login).
			WillReturnError(sql.ErrNoRows)

		user, err := store.GetUserByLogin(ctx, login)
		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("SaveOrder - New Order", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "123456789"

		mock.ExpectQuery("SELECT user_id FROM orders WHERE number").
			WithArgs(orderNumber).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectExec("INSERT INTO orders").
			WithArgs(orderNumber, userID, models.StatusRegistered).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.SaveOrder(ctx, userID, orderNumber)
		assert.NoError(t, err)
	})

	t.Run("SaveOrder - Order Exists for Another User", func(t *testing.T) {
		userID := int64(2)
		orderNumber := "123456789"
		existingUserID := sql.NullInt64{Int64: 1, Valid: true}

		mock.ExpectQuery("SELECT user_id FROM orders WHERE number").
			WithArgs(orderNumber).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(existingUserID))

		err := store.SaveOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, apperrors.ErrOrderExistsForOther)
	})

	t.Run("GetBalance - Success", func(t *testing.T) {
		userID := int64(1)
		expectedBalance := models.Balance{Current: 100.50, Withdrawn: 50.25}

		rows := sqlmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(expectedBalance.Current, expectedBalance.Withdrawn)
		mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id").
			WithArgs(userID).
			WillReturnRows(rows)

		balance, err := store.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance.Current, balance.Current)
		assert.Equal(t, expectedBalance.Withdrawn, balance.Withdrawn)
	})

	t.Run("UpdateBalance - Insufficient Funds", func(t *testing.T) {
		userID := int64(1)
		currentBalance := 10.0
		delta := -20.0

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT current FROM balances WHERE user_id").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"current"}).AddRow(currentBalance))
		mock.ExpectRollback()

		err := store.UpdateBalance(ctx, userID, delta)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("CreateUser - Success", func(t *testing.T) {
		login := "test_user"
		passwordHash := "hashed_password"
		userID := int64(1)

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(login, passwordHash).
			WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "created_at"}).
				AddRow(userID, login, passwordHash, time.Now()))

		mock.ExpectExec("INSERT INTO balances").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		user, err := store.CreateUser(ctx, login, passwordHash)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, login, user.Login)
	})

	t.Run("SaveOrder - Success", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "123456789"

		mock.ExpectQuery("SELECT user_id FROM orders WHERE number").
			WithArgs(orderNumber).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectExec("INSERT INTO orders").
			WithArgs(orderNumber, userID, models.StatusRegistered).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.SaveOrder(ctx, userID, orderNumber)
		assert.NoError(t, err)
	})

	t.Run("GetOrder - Order Exists", func(t *testing.T) {
		orderNumber := "123456789"
		order := models.Order{ID: 1, Number: orderNumber, UserID: 1, Status: models.StatusRegistered}

		rows := sqlmock.NewRows([]string{"id", "number", "user_id", "status", "accrual", "uploaded_at"}).
			AddRow(order.ID, order.Number, order.UserID, order.Status, sql.NullFloat64{}, time.Now())
		mock.ExpectQuery("SELECT id, number, user_id, status, accrual, uploaded_at FROM orders").
			WithArgs(orderNumber).
			WillReturnRows(rows)

		result, err := store.GetOrder(ctx, orderNumber)
		assert.NoError(t, err)
		assert.Equal(t, order.Number, result.Number)
	})

	t.Run("GetBalance - Success", func(t *testing.T) {
		userID := int64(1)
		expectedBalance := models.Balance{Current: 100.50, Withdrawn: 50.25}

		rows := sqlmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(expectedBalance.Current, expectedBalance.Withdrawn)
		mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id").
			WithArgs(userID).
			WillReturnRows(rows)

		balance, err := store.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance.Current, balance.Current)
		assert.Equal(t, expectedBalance.Withdrawn, balance.Withdrawn)
	})

	t.Run("UpdateBalance - Success", func(t *testing.T) {
		userID := int64(1)
		delta := 10.0
		currentBalance := 20.0

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT current FROM balances WHERE user_id").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"current"}).AddRow(currentBalance))
		mock.ExpectExec("UPDATE balances SET current = current +").
			WithArgs(delta, userID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := store.UpdateBalance(ctx, userID, delta)
		assert.NoError(t, err)
	})

	t.Run("CreateWithdrawal - Success", func(t *testing.T) {
		userID := int64(1)
		orderNumber := "order123"
		sum := 50.0
		currentBalance := 100.0

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT current FROM balances WHERE user_id").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"current"}).AddRow(currentBalance))
		mock.ExpectExec("INSERT INTO withdrawals").
			WithArgs(userID, orderNumber, sum).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE balances SET current = current -").
			WithArgs(sum, userID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := store.CreateWithdrawal(ctx, userID, orderNumber, sum)
		assert.NoError(t, err)
	})

	t.Run("GetWithdrawals - Success", func(t *testing.T) {
		userID := int64(1)
		expectedWithdrawals := []models.Withdrawal{
			{ID: 1, UserID: userID, OrderNumber: "order123", Sum: 50.0, ProcessedAt: fixedTime},
		}

		rows := sqlmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"})
		for _, w := range expectedWithdrawals {
			rows.AddRow(w.ID, w.UserID, w.OrderNumber, w.Sum, fixedTime)
		}
		mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals").
			WithArgs(userID).
			WillReturnRows(rows)

		withdrawals, err := store.GetWithdrawals(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedWithdrawals, withdrawals)
	})

	t.Run("UpdateOrderStatus - Success", func(t *testing.T) {
		orderNumber := "123456789"
		status := models.StatusProcessed
		accrual := 15.75

		mock.ExpectExec("UPDATE orders SET status").
			WithArgs(status, accrual, orderNumber).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.UpdateOrderStatus(ctx, orderNumber, status, accrual)
		assert.NoError(t, err)
	})

	t.Run("GetUserOrders - Success", func(t *testing.T) {
		userID := int64(1)
		expectedOrders := []models.Order{
			{ID: 1, Number: "123456", UserID: userID, Status: models.StatusRegistered, UploadedAt: fixedTime},
		}

		rows := sqlmock.NewRows([]string{"id", "number", "user_id", "status", "accrual", "uploaded_at"})
		for _, order := range expectedOrders {
			rows.AddRow(order.ID, order.Number, order.UserID, order.Status, sql.NullFloat64{}, fixedTime)
		}
		mock.ExpectQuery("SELECT id, number, user_id, status, accrual, uploaded_at FROM orders WHERE user_id").
			WithArgs(userID).
			WillReturnRows(rows)

		orders, err := store.GetUserOrders(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrders, orders)
	})

	t.Run("Ping - Success", func(t *testing.T) {
		mock.ExpectPing()

		err := store.Ping(ctx)
		assert.NoError(t, err)
	})
}
