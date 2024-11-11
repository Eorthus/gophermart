package accrual

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Eorthus/gophermart/internal/models"
)

// RateLimitError представляет ошибку превышения лимита запросов
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
}

// OrderNotFoundError представляет ошибку отсутствия заказа
type OrderNotFoundError struct {
	OrderNumber string
}

func (e *OrderNotFoundError) Error() string {
	return fmt.Sprintf("order %s not found", e.OrderNumber)
}

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

func (c *Client) GetOrderAccrual(orderNumber string) (*models.AccrualResponse, error) {
	if orderNumber == "" {
		return nil, fmt.Errorf("order number cannot be empty")
	}

	// Правильно формируем URL
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var accrual models.AccrualResponse
		if err := json.Unmarshal(body, &accrual); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &accrual, nil

	case http.StatusNoContent:
		return nil, nil

	case http.StatusNotFound:
		// Возможно, неправильный путь
		return nil, fmt.Errorf("check accrual system URL configuration: %s returned 404", url)

	default:
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
}
