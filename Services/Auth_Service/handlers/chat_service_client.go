// Auth_Service/handlers/chat_service_client.go
package handlers

import (
	"Auth_Service/config"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatServiceClient клиент для взаимодействия с Chat Service
type ChatServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewChatServiceClient создает новый клиент Chat Service
func NewChatServiceClient(cfg config.ChatServiceConfig) *ChatServiceClient {
	return &ChatServiceClient{
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

// DeleteChatMembersByUserID удаляет все записи пользователя из chat_members
func (c *ChatServiceClient) DeleteChatMembersByUserID(ctx context.Context, userID int) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/internal/members/%d", c.baseURL, userID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
