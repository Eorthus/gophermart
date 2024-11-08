package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/Eorthus/gophermart/internal/apperrors"
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
	passwordHash := hashPassword(password)
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

	if user.PasswordHash != hashPassword(password) {
		return nil, apperrors.ErrInvalidCredentials
	}

	return user, nil
}

func hashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}
