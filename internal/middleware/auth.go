package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/Eorthus/gophermart/internal/utils"
	"go.uber.org/zap"
)

const (
	cookieName = "auth_token"
	secretKey  = "your-secret-key" // В реальном приложении следует использовать безопасное хранение ключа
)

func AuthMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("Processing request through auth middleware",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method))

			userID := GetUserID(r)
			if userID == "" {
				logger.Debug("No valid user ID found in request")
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized"))
				return
			}

			logger.Debug("Request authenticated",
				zap.String("userID", userID))
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID извлекает ID пользователя из cookie
func GetUserID(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		// Добавляем логирование для отладки
		log.Printf("No auth cookie found: %v", err)
		return ""
	}

	parts := utils.SplitString(cookie.Value, ":")
	if len(parts) != 2 {
		log.Printf("Invalid cookie format: %s", cookie.Value)
		return ""
	}

	userID, signature := parts[0], parts[1]
	if !isValidSignature(userID, signature) {
		log.Printf("Invalid signature for user ID: %s", userID)
		return ""
	}

	return userID
}

// SetAuthCookie устанавливает cookie с ID пользователя
func SetAuthCookie(w http.ResponseWriter, userID string) {
	signature := generateSignature(userID)
	value := userID + ":" + signature
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Вспомогательные функции
func generateSignature(data string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

func isValidSignature(data, signature string) bool {
	return generateSignature(data) == signature
}
