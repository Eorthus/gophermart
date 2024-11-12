package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eorthus/gophermart/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// Вспомогательная функция для установки тестового cookie
func setAuthCookieForTest(req *http.Request, userID string) {
	signature := generateSignature(userID)
	value := userID + ":" + signature
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: value,
	})
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupAuth      func(*http.Request)
		expectedStatus int
	}{
		{
			name: "valid auth cookie",
			setupAuth: func(req *http.Request) {
				setAuthCookieForTest(req, "123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no cookie",
			setupAuth:      func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid cookie format",
			setupAuth: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  cookieName,
					Value: "invalid-cookie",
				})
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid signature",
			setupAuth: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  cookieName,
					Value: "123:wrong-signature",
				})
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)

			// Создаём тестовый обработчик
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(logger)(nextHandler)

			// Создаём тестовый запрос
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupAuth(req)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name       string
		setupAuth  func(*http.Request)
		expectedID string
	}{
		{
			name: "valid cookie",
			setupAuth: func(req *http.Request) {
				setAuthCookieForTest(req, "123")
			},
			expectedID: "123",
		},
		{
			name:       "no cookie",
			setupAuth:  func(req *http.Request) {},
			expectedID: "",
		},
		{
			name: "invalid cookie format",
			setupAuth: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  cookieName,
					Value: "invalid-cookie",
				})
			},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupAuth(req)

			gotID := GetUserID(req)
			assert.Equal(t, tt.expectedID, gotID)
		})
	}
}

func TestSetAuthCookie(t *testing.T) {
	tests := []struct {
		name   string
		userID string
	}{
		{
			name:   "normal user id",
			userID: "123",
		},
		{
			name:   "empty user id",
			userID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			// Устанавливаем cookie
			SetAuthCookie(rr, tt.userID)

			// Получаем установленные cookies
			res := rr.Result()     // Получаем результат один раз
			defer res.Body.Close() // Закрываем тело ответа после получения

			cookies := res.Cookies() // Используем переменную `res`

			// Должен быть установлен ровно один cookie
			assert.Len(t, cookies, 1)

			if len(cookies) > 0 {
				cookie := cookies[0]
				assert.Equal(t, cookieName, cookie.Name)
				assert.True(t, cookie.HttpOnly)

				// Проверяем формат значения cookie
				parts := utils.SplitString(cookie.Value, ":")
				assert.Len(t, parts, 2)

				if len(parts) == 2 {
					assert.Equal(t, tt.userID, parts[0])
					assert.True(t, isValidSignature(parts[0], parts[1]))
				}
			}
		})
	}
}

func TestGenerateAndValidateSignature(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "normal string",
			data: "test-data",
		},
		{
			name: "empty string",
			data: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Генерируем подпись
			signature := generateSignature(tt.data)

			// Проверяем, что подпись не пустая
			assert.NotEmpty(t, signature)

			// Проверяем валидность подписи
			assert.True(t, isValidSignature(tt.data, signature))

			// Проверяем невалидную подпись
			assert.False(t, isValidSignature(tt.data, "invalid-signature"))
		})
	}
}
