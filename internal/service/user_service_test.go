// service/user_service_test.go
package service

import (
	"context"
	"testing"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/Eorthus/gophermart/internal/storage"
	"github.com/Eorthus/gophermart/internal/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_RegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock.NewMockStorage(ctrl)
	service := NewUserService(mockStorage)
	ctx := context.Background()

	tests := []struct {
		name        string
		login       string
		password    string
		mockSetup   func()
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "successful registration",
			login:    "testuser",
			password: "testpass",
			mockSetup: func() {
				mockStorage.EXPECT().
					CreateUser(ctx, "testuser", gomock.Any()). // gomock.Any() для хеша пароля
					Return(&models.User{
						ID:           1,
						Login:        "testuser",
						PasswordHash: "hashed_password",
					}, nil)
			},
			wantErr: false,
		},
		{
			name:     "duplicate user",
			login:    "existing",
			password: "testpass",
			mockSetup: func() {
				mockStorage.EXPECT().
					CreateUser(ctx, "existing", gomock.Any()).
					Return(nil, storage.ErrUserExists)
			},
			wantErr:     true,
			expectedErr: storage.ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			user, err := service.RegisterUser(ctx, tt.login, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, user)
			assert.Equal(t, tt.login, user.Login)
		})
	}
}

func TestUserService_AuthenticateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock.NewMockStorage(ctrl)
	service := NewUserService(mockStorage)
	ctx := context.Background()

	// Создаем тестового пользователя с известным хешем пароля
	hashedPassword := middleware.HashPassword("correctpass")
	testUser := &models.User{
		ID:           1,
		Login:        "testuser",
		PasswordHash: hashedPassword,
	}

	tests := []struct {
		name        string
		login       string
		password    string
		mockSetup   func()
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "successful authentication",
			login:    "testuser",
			password: "correctpass",
			mockSetup: func() {
				mockStorage.EXPECT().
					GetUserByLogin(ctx, "testuser").
					Return(testUser, nil)
			},
			wantErr: false,
		},
		{
			name:     "wrong password",
			login:    "testuser",
			password: "wrongpass",
			mockSetup: func() {
				mockStorage.EXPECT().
					GetUserByLogin(ctx, "testuser").
					Return(testUser, nil)
			},
			wantErr:     true,
			expectedErr: apperrors.ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			login:    "nonexistent",
			password: "anypass",
			mockSetup: func() {
				mockStorage.EXPECT().
					GetUserByLogin(ctx, "nonexistent").
					Return(nil, storage.ErrUserNotFound)
			},
			wantErr:     true,
			expectedErr: apperrors.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			user, err := service.AuthenticateUser(ctx, tt.login, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, user)
			assert.Equal(t, tt.login, user.Login)
			assert.Equal(t, testUser.ID, user.ID)
		})
	}
}
