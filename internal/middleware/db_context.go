package middleware

import (
	"context"
	"net/http"

	"github.com/Eorthus/gophermart/internal/storage"
)

// Определяем тип для ключа контекста
type contextKey string

// Создаем константу для ключа контекста
const dbContextKey contextKey = "db"

func GetDBFromContext(ctx context.Context) (storage.Storage, bool) {
	store, ok := ctx.Value(dbContextKey).(storage.Storage)
	return store, ok
}

func DBContextMiddleware(store storage.Storage) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), dbContextKey, store)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
