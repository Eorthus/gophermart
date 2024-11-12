package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApiContextMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		handlerDelay      time.Duration
		middlewareTimeout time.Duration
		expectedStatus    int
	}{
		{
			name:              "Request completes within timeout",
			handlerDelay:      50 * time.Millisecond,
			middlewareTimeout: 100 * time.Millisecond,
			expectedStatus:    http.StatusOK,
		},
		{
			name:              "Request times out",
			handlerDelay:      150 * time.Millisecond,
			middlewareTimeout: 100 * time.Millisecond,
			expectedStatus:    http.StatusGatewayTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.handlerDelay)
				w.WriteHeader(http.StatusOK)
			})

			middleware := APIContextMiddleware(tt.middlewareTimeout)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "http://example.com", nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
