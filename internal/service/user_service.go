package service

import (
	"context"
	"strings"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
)

type UserService struct {
	store storage.Storage
}

func NewUserService(store storage.Storage) *UserService {
	return &UserService{store: store}
}

func (s *UserService) RegisterUser(ctx context.Context, login, password string) (*models.User, error) {
	passwordHash := middleware.HashPassword(password)
	user, err := s.store.CreateUser(ctx, login, passwordHash)
	if err != nil {
		// Проверяем ошибку на нарушение уникальности
		if strings.Contains(err.Error(), "users_login_key") {
			return nil, apperrors.ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) AuthenticateUser(ctx context.Context, login, password string) (*models.User, error) {
	user, err := s.store.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if user.PasswordHash != middleware.HashPassword(password) {
		return nil, apperrors.ErrInvalidCredentials
	}

	return user, nil
}
