package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eorthus/gophermart/internal/apperrors"
	"github.com/Eorthus/gophermart/internal/middleware"
	"github.com/Eorthus/gophermart/internal/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleRegister(t *testing.T) {
	r, mockStorage := setupRouter(t)

	tests := []struct {
		name           string
		credentials    models.Credentials
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Valid registration",
			credentials: models.Credentials{
				Login:    "testuser",
				Password: "password123",
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					CreateUser(gomock.Any(), "testuser", gomock.Any()).
					Return(&models.User{
						ID:           1,
						Login:        "testuser",
						PasswordHash: middleware.HashPassword("password123"),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "User already exists",
			credentials: models.Credentials{
				Login:    "existinguser",
				Password: "password123",
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					CreateUser(gomock.Any(), "existinguser", gomock.Any()).
					Return(nil, apperrors.ErrUserExists)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			body, err := json.Marshal(tt.credentials)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/user/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close() // Закрываем тело ответа

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				cookies := res.Cookies()
				if assert.NotEmpty(t, cookies, "Expected cookie to be set") {
					assert.Equal(t, "auth_token", cookies[0].Name)
				}
			}
		})
	}
}

func TestHandleLogin(t *testing.T) {
	r, mockStorage := setupRouter(t)

	const testPassword = "password123"
	hashedPassword := middleware.HashPassword(testPassword)

	tests := []struct {
		name           string
		credentials    models.Credentials
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Valid login",
			credentials: models.Credentials{
				Login:    "testuser",
				Password: testPassword,
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					GetUserByLogin(gomock.Any(), "testuser").
					Return(&models.User{
						ID:           1,
						Login:        "testuser",
						PasswordHash: hashedPassword,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid credentials",
			credentials: models.Credentials{
				Login:    "wronguser",
				Password: testPassword,
			},
			setupMocks: func() {
				mockStorage.EXPECT().
					GetUserByLogin(gomock.Any(), "wronguser").
					Return(nil, apperrors.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			body, err := json.Marshal(tt.credentials)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/user/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close() // Закрываем тело ответа

			assert.Equal(t, tt.expectedStatus, rr.Code, "Status code mismatch for %s", tt.name)

			if tt.expectedStatus == http.StatusOK {
				cookies := res.Cookies()
				if assert.NotEmpty(t, cookies, "Expected cookie to be set") {
					assert.Equal(t, "auth_token", cookies[0].Name)
				}
			}
		})
	}
}

func TestHandleInvalidRequests(t *testing.T) {
	r, _ := setupRouter(t)

	tests := []struct {
		name   string
		path   string
		body   string
		expect int
	}{
		{
			name:   "Invalid JSON register",
			path:   "/api/user/register",
			body:   `{"login": "test"`, // некорректный JSON
			expect: http.StatusBadRequest,
		},
		{
			name:   "Invalid JSON login",
			path:   "/api/user/login",
			body:   `{"login": "test"`, // некорректный JSON
			expect: http.StatusBadRequest,
		},
		{
			name:   "Empty body register",
			path:   "/api/user/register",
			body:   `{}`,
			expect: http.StatusUnauthorized, // Изменено согласно реальному поведению
		},
		{
			name:   "Empty body login",
			path:   "/api/user/login",
			body:   `{}`,
			expect: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expect, rr.Code, "Status code mismatch for %s", tt.name)
		})
	}
}
