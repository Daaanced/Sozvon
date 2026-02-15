// Chat_Service/models/models.go
package models

import (
	"errors"
	"time"
)

var (
	ErrInvalidChatID  = errors.New("invalid chat ID")
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrMessageTooLong = errors.New("message too long")
	ErrInvalidMembers = errors.New("invalid chat members")
)

const (
	MaxMessageLength = 4000 // Максимальная длина сообщения
)

// Chat представляет чат
type Chat struct {
	ID        string    `json:"id"`
	Members   []string  `json:"members"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatListItem элемент списка чатов
type ChatListItem struct {
	ChatID      string    `json:"chatId"`
	Members     []string  `json:"members"`
	LastMessage string    `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Message представляет сообщение
type Message struct {
	ID          string    `json:"id"`
	ChatID      string    `json:"chatId"`
	SenderLogin string    `json:"from"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateChatRequest запрос на создание чата
type CreateChatRequest struct {
	From string `json:"from" validate:"required"`
	To   string `json:"to" validate:"required"`
}

// Validate валидирует запрос создания чата
func (r *CreateChatRequest) Validate() error {
	if r.From == "" || r.To == "" {
		return ErrInvalidMembers
	}
	if r.From == r.To {
		return errors.New("cannot create chat with yourself")
	}
	return nil
}

// SendMessageRequest запрос на отправку сообщения
type SendMessageRequest struct {
	ChatID string `json:"chatId"`
	Text   string `json:"text"`
}

// Validate валидирует запрос отправки сообщения
func (r *SendMessageRequest) Validate() error {
	if r.ChatID == "" {
		return ErrInvalidChatID
	}
	if r.Text == "" {
		return ErrEmptyMessage
	}
	if len(r.Text) > MaxMessageLength {
		return ErrMessageTooLong
	}
	return nil
}

// WSMessage структура WebSocket сообщения
type WSMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// WSMessageData данные WebSocket сообщения
type WSMessageData struct {
	ChatID string `json:"chatId"`
	Text   string `json:"text"`
	From   string `json:"from,omitempty"`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse структура успешного ответа
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
