package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/service"
	"go.uber.org/zap"
)

type OrderHandler struct {
	orderService *service.OrderService
	logger       *zap.Logger
}

func NewOrderHandler(orderService *service.OrderService, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// HandleSubmitOrder обрабатывает загрузку номера заказа
func (h *OrderHandler) HandleSubmitOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(middleware.GetUserID(r), 10, 64)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrUnauthorized, h.logger)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrInvalidRequestFormat, h.logger)
		return
	}

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		apperrors.HandleError(w, err, h.logger)
		return
	}

	err = h.orderService.SubmitOrder(r.Context(), userID, orderNumber)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidOrder):
			apperrors.HandleError(w, apperrors.ErrInvalidOrder, h.logger)
		case errors.Is(err, apperrors.ErrOrderExistsForUser):
			w.WriteHeader(http.StatusOK)
		case errors.Is(err, apperrors.ErrOrderExistsForOther):
			apperrors.HandleError(w, apperrors.ErrOrderExistsForOther, h.logger)
		default:
			apperrors.HandleError(w, err, h.logger)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// HandleGetOrders возвращает список заказов пользователя
func (h *OrderHandler) HandleGetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(middleware.GetUserID(r), 10, 64)
	if err != nil {
		apperrors.HandleError(w, apperrors.ErrUnauthorized, h.logger)
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), userID)
	if err != nil {
		apperrors.HandleError(w, err, h.logger)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		apperrors.HandleError(w, err, h.logger)
	}
}
