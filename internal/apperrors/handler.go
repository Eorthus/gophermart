package apperrors

import (
	"errors"
	"net/http"

	"go.uber.org/zap"
)

func HandleError(w http.ResponseWriter, err error, logger *zap.Logger) {
	var appErr AppError
	if errors.As(err, &appErr) {
		if appErr.Status >= 500 {
			logger.Error("Internal server error",
				zap.Error(err),
				zap.Int("status", appErr.Status),
			)
		} else {
			logger.Info("Client error",
				zap.Error(err),
				zap.Int("status", appErr.Status),
			)
		}

		// Для статуса 200 не отправляем тело ответа
		if appErr.Status == http.StatusOK {
			w.WriteHeader(appErr.Status)
			return
		}

		http.Error(w, appErr.Message, appErr.Status)
		return
	}

	// Если ошибка не является AppError, логируем как внутреннюю ошибку
	logger.Error("Unexpected error",
		zap.Error(err),
	)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
