package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	t.Run("Сжатие ответа", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()

		GzipMiddleware(handler).ServeHTTP(rr, req)

		assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

		gr, err := gzip.NewReader(rr.Body)
		require.NoError(t, err)
		defer gr.Close()

		body, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))
	})

	t.Run("Распаковка запроса", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(body)
		})

		content := "compressed request body"
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		gw.Write([]byte(content))
		gw.Close()

		req := httptest.NewRequest("POST", "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		rr := httptest.NewRecorder()

		GzipMiddleware(handler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, content, rr.Body.String())
	})

	t.Run("Без сжатия, когда клиент не поддерживает", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		})

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		GzipMiddleware(handler).ServeHTTP(rr, req)

		assert.Empty(t, rr.Header().Get("Content-Encoding"))
		assert.Equal(t, "hello world", rr.Body.String())
	})

	t.Run("Без распаковки для несжатого запроса", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(body)
		})

		content := "uncompressed request body"
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
		rr := httptest.NewRecorder()

		GzipMiddleware(handler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, content, rr.Body.String())
	})
}
