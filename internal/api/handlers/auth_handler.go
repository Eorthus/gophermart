package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

func NewAuthHandler(userService *service.UserService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:      logger,
	}
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		apperrors.HandleError(w, apperrors.ErrInvalidRequestFormat, h.logger)
		return
	}

	// Валидация формата запроса
	if err := validateCredentials(creds); err != nil {
		apperrors.HandleError(w, apperrors.ErrInvalidCredentials, h.logger)
		return
	}

	user, err := h.userService.RegisterUser(r.Context(), creds.Login, creds.Password)
	if err != nil {
		if err == apperrors.ErrUserExists {
			apperrors.HandleError(w, apperrors.ErrUserExists, h.logger)
			return
		}
		apperrors.HandleError(w, err, h.logger)
		return
	}

	// После успешной регистрации автоматически аутентифицируем пользователя
	middleware.SetAuthCookie(w, strconv.FormatInt(user.ID, 10))
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var creds models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		apperrors.HandleError(w, apperrors.ErrInvalidRequestFormat, h.logger)
		return
	}

	// Проверяем наличие обязательных полей
	if creds.Login == "" || creds.Password == "" {
		apperrors.HandleError(w, apperrors.ErrInvalidRequestFormat, h.logger)
		return
	}

	// Пытаемся аутентифицировать пользователя
	user, err := h.userService.AuthenticateUser(r.Context(), creds.Login, creds.Password)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			apperrors.HandleError(w, apperrors.ErrInvalidCredentials, h.logger)
			return
		}
		apperrors.HandleError(w, err, h.logger)
		return
	}

	// Успешная аутентификация
	middleware.SetAuthCookie(w, strconv.FormatInt(user.ID, 10))
	w.WriteHeader(http.StatusOK)
}

// validateCredentials проверяет формат учетных данных
func validateCredentials(creds models.Credentials) error {
	if creds.Login == "" {
		return fmt.Errorf("login is required")
	}
	if len(creds.Login) < 3 || len(creds.Login) > 50 {
		return fmt.Errorf("login must be between 3 and 50 characters")
	}
	if creds.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(creds.Password) < 6 || len(creds.Password) > 50 {
		return fmt.Errorf("password must be between 6 and 50 characters")
	}
	return nil
}
