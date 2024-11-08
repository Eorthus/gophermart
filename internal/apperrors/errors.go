package apperrors

import (
	"net/http"
)

type AppError struct {
	Status  int
	Message string
	Err     error
}

func (e AppError) Error() string {
	return e.Message
}

func (e AppError) Unwrap() error {
	return e.Err
}

var (
	// Auth errors
	ErrUserExists         = AppError{Status: http.StatusConflict, Message: "user already exists"}
	ErrInvalidCredentials = AppError{Status: http.StatusUnauthorized, Message: "invalid credentials"}
	ErrUnauthorized       = AppError{Status: http.StatusUnauthorized, Message: "unauthorized"}

	// Order errors
	ErrInvalidOrder        = AppError{Status: http.StatusUnprocessableEntity, Message: "Invalid order number: Luhn algorithm check failed"}
	ErrOrderExistsForUser  = AppError{Status: http.StatusOK, Message: "order already exists for this user"}
	ErrOrderExistsForOther = AppError{Status: http.StatusConflict, Message: "order already uploaded by another user"}
	ErrOrderNotFound       = AppError{Status: http.StatusNotFound, Message: "order not found"}

	// Balance errors
	ErrInsufficientFunds = AppError{Status: http.StatusPaymentRequired, Message: "insufficient funds"}
	ErrInvalidWithdraw   = AppError{Status: http.StatusUnprocessableEntity, Message: "invalid withdraw amount"}

	// Request errors
	ErrInvalidRequestFormat = AppError{Status: http.StatusBadRequest, Message: "invalid request format"}
)
