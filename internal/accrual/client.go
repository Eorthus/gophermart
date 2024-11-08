package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Eorthus/gophermart/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrderAccrual получает информацию о начислении баллов за заказ
func (c *Client) GetOrderAccrual(orderNumber string) (*models.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrual models.AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrual); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &accrual, nil

	case http.StatusNoContent:
		return nil, nil

	case http.StatusTooManyRequests:
		// Получаем время ожидания из заголовка
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			seconds, err := time.ParseDuration(retryAfter + "s")
			if err == nil {
				return nil, &RateLimitError{RetryAfter: seconds}
			}
		}
		return nil, &RateLimitError{RetryAfter: 60 * time.Second} // По умолчанию ждем 60 секунд

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// RateLimitError представляет ошибку превышения лимита запросов
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
}
