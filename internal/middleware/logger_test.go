package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// Общая функция для проверки GET-запросов
func checkGETLog(t *testing.T, logEntry observer.LoggedEntry, uri string) {
	assert.Equal(t, "GET Request", logEntry.Message)
	assert.Equal(t, uri, logEntry.ContextMap()["uri"])
	assert.Equal(t, "GET", logEntry.ContextMap()["method"])

	duration, ok := logEntry.ContextMap()["duration"].(time.Duration)
	assert.True(t, ok)
	assert.Greater(t, duration, time.Duration(0))

	assert.NotContains(t, logEntry.ContextMap(), "status")
	assert.NotContains(t, logEntry.ContextMap(), "size")
}

// Общая функция для проверки POST-запросов
func checkPOSTLog(t *testing.T, logEntry observer.LoggedEntry, statusCode int64, size int64) {
	assert.Equal(t, "POST Response", logEntry.Message)
	assert.Equal(t, statusCode, logEntry.ContextMap()["status"])
	assert.Equal(t, size, logEntry.ContextMap()["size"])

	assert.NotContains(t, logEntry.ContextMap(), "uri")
	assert.NotContains(t, logEntry.ContextMap(), "method")
	assert.NotContains(t, logEntry.ContextMap(), "duration")
}

func TestGETLogger(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// Тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond) // Добавляем небольшую задержку
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Оборачиваем тестовый обработчик GETLogger'ом
	handler := GETLogger(logger)(testHandler)

	t.Run("GET request", func(t *testing.T) {
		t.Parallel()
		// Очищаем логи перед каждым тестом
		recorded.TakeAll()

		req := httptest.NewRequest("GET", "/test-get", nil)
		rec := httptest.NewRecorder()

		// Выполняем запрос
		handler.ServeHTTP(rec, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rec.Code)

		// Проверяем, что лог был записан
		assert.Equal(t, 1, recorded.Len())
		logEntry := recorded.All()[0]

		// Проверяем поля лога для GET запроса
		checkGETLog(t, logEntry, "/test-get")
	})
}

func TestPOSTLogger(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// Тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // 201 статус для POST
		w.Write([]byte("OK"))
	})

	// Оборачиваем тестовый обработчик POSTLogger'ом
	handler := POSTLogger(logger)(testHandler)

	t.Run("POST request", func(t *testing.T) {
		t.Parallel()
		// Очищаем логи перед каждым тестом
		recorded.TakeAll()

		req := httptest.NewRequest("POST", "/test-post", nil)
		rec := httptest.NewRecorder()

		// Выполняем запрос
		handler.ServeHTTP(rec, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Проверяем, что лог был записан
		assert.Equal(t, 1, recorded.Len())
		logEntry := recorded.All()[0]

		// Проверяем поля лога для POST запроса
		checkPOSTLog(t, logEntry, 201, 2) // 2 байта — это "OK"
	})
}

func TestLogger(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// Тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Оборачиваем тестовый обработчик общим логгером
	handler := Logger(logger)(testHandler)

	t.Run("GET request", func(t *testing.T) {
		t.Parallel()
		// Очищаем логи перед каждым тестом
		recorded.TakeAll()

		req := httptest.NewRequest("GET", "/test-get", nil)
		rec := httptest.NewRecorder()

		// Выполняем запрос
		handler.ServeHTTP(rec, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rec.Code)

		// Проверяем, что лог был записан
		assert.Equal(t, 1, recorded.Len())
		logEntry := recorded.All()[0]

		// Общий лог должен логировать все запросы, проверяем поля для GET
		assert.Equal(t, "Request", logEntry.Message)
		assert.Equal(t, "/test-get", logEntry.ContextMap()["uri"])
		assert.Equal(t, "GET", logEntry.ContextMap()["method"])
	})

	t.Run("POST request", func(t *testing.T) {
		t.Parallel()
		// Очищаем логи перед каждым тестом
		recorded.TakeAll()

		req := httptest.NewRequest("POST", "/test-post", nil)
		rec := httptest.NewRecorder()

		// Выполняем запрос
		handler.ServeHTTP(rec, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rec.Code)

		// Проверяем, что лог был записан
		assert.Equal(t, 1, recorded.Len())
		logEntry := recorded.All()[0]

		// Общий лог должен логировать все запросы, проверяем поля для POST
		assert.Equal(t, "Request", logEntry.Message)
		assert.Equal(t, "/test-post", logEntry.ContextMap()["uri"])
		assert.Equal(t, "POST", logEntry.ContextMap()["method"])
	})
}
