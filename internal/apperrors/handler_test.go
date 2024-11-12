package apperrors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestHandleError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedBody string
	}{
		{
			name:         "user exists error",
			err:          ErrUserExists,
			expectedCode: http.StatusConflict,
			expectedBody: "user already exists\n",
		},
		{
			name:         "invalid credentials error",
			err:          ErrInvalidCredentials,
			expectedCode: http.StatusUnauthorized,
			expectedBody: "invalid credentials\n",
		},
		{
			name:         "unauthorized error",
			err:          ErrUnauthorized,
			expectedCode: http.StatusUnauthorized,
			expectedBody: "unauthorized\n",
		},
		{
			name:         "invalid order error",
			err:          ErrInvalidOrder,
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: "Invalid order number: Luhn algorithm check failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger := zaptest.NewLogger(t)

			HandleError(w, tt.err, logger)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
