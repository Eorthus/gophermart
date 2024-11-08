package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/service"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	balanceService *service.BalanceService
	logger         *zap.Logger
}

func NewBalanceHandler(balanceService *service.BalanceService, logger *zap.Logger) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
		logger:         logger,
	}
}

// HandleGetBalance возвращает текущий баланс пользователя
func (h *BalanceHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(middleware.GetUserID(r), 10, 64)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrUnauthorized, h.logger)
		return
	}

	balance, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		apperrors.HandleError(w, err, h.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		apperrors.HandleError(w, err, h.logger)
	}
}

// HandleWithdraw обрабатывает запрос на списание средств
func (h *BalanceHandler) HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(middleware.GetUserID(r), 10, 64)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrUnauthorized, h.logger)
		return
	}

	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.HandleError(w, apperrors.ErrInvalidRequestFormat, h.logger)
		return
	}

	err = h.balanceService.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch err {
		case apperrors.ErrInvalidOrder:
			apperrors.HandleError(w, apperrors.ErrInvalidOrder, h.logger)
		case apperrors.ErrInsufficientFunds:
			apperrors.HandleError(w, apperrors.ErrInsufficientFunds, h.logger)
		default:
			apperrors.HandleError(w, err, h.logger)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetWithdrawals возвращает историю списаний
func (h *BalanceHandler) HandleGetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(middleware.GetUserID(r), 10, 64)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrUnauthorized, h.logger)
		return
	}

	withdrawals, err := h.balanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		apperrors.HandleError(w, err, h.logger)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		apperrors.HandleError(w, err, h.logger)
	}
}
