// Authorization/handlers/user_service_client.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"Authorization/config"
)

// UserServiceClient клиент для взаимодействия с User Service
type UserServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewUserServiceClient создает новый клиент User Service
func NewUserServiceClient(cfg config.UserServiceConfig) *UserServiceClient {
	return &UserServiceClient{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// userProfileRequest структура для создания профиля пользователя
type userProfileRequest struct {
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Info    string `json:"info"`
	Picture string `json:"picture"`
}

// CreateUserProfile создает профиль пользователя в User Service
func (c *UserServiceClient) CreateUserProfile(ctx context.Context, login string) error {
	body := userProfileRequest{
		Login:   login,
		Name:    login,
		Email:   "",
		Info:    "",
		Picture: "",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Попытка создания с повторами
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := c.createUserProfileAttempt(ctx, jsonBody); err != nil {
			lastErr = err
			if attempt < maxRetries {
				// Экспоненциальная задержка: 100ms, 200ms, 400ms
				backoff := time.Duration(100*attempt*attempt) * time.Millisecond
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to create user profile after %d attempts: %w", maxRetries, lastErr)
}

// createUserProfileAttempt выполняет одну попытку создания профиля
func (c *UserServiceClient) createUserProfileAttempt(ctx context.Context, jsonBody []byte) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/users",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для логирования
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("user service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteUserProfile удаляет профиль пользователя (для отката транзакций)
func (c *UserServiceClient) DeleteUserProfile(ctx context.Context, login string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/users/%s", c.baseURL, login),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HealthCheck проверяет доступность User Service
func (c *UserServiceClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		c.baseURL+"/health",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// --- Обратная совместимость ---

// createUserProfile старая функция для обратной совместимости
func createUserProfile(login string) error {
	ctx := context.Background()
	cfg := config.UserServiceConfig{
		URL:     "http://localhost:8083",
		Timeout: 10 * time.Second,
	}
	client := NewUserServiceClient(cfg)
	return client.CreateUserProfile(ctx, login)
}
