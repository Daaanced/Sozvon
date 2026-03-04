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

const MaxMessageLength = 4000

type Chat struct {
	ID        string    `json:"id"`
	Members   []int     `json:"members"` // user IDs
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatListItem struct {
	ChatID      string    `json:"chatId"`
	Members     []int     `json:"members"` // user IDs
	LastMessage string    `json:"lastMessage"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chatId"`
	SenderID  int       `json:"senderId"` // user ID
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateChatRequest — from/to теперь user_id
type CreateChatRequest struct {
	FromID int `json:"from_id"`
	ToID   int `json:"to_id"`
}

func (r *CreateChatRequest) Validate() error {
	if r.FromID == 0 || r.ToID == 0 {
		return ErrInvalidMembers
	}
	if r.FromID == r.ToID {
		return errors.New("cannot create chat with yourself")
	}
	return nil
}

// SendMessageRequest — REST тело для отправки сообщения
type SendMessageRequest struct {
	Text string `json:"text"`
}

func (r *SendMessageRequest) Validate() error {
	if r.Text == "" {
		return ErrEmptyMessage
	}
	if len(r.Text) > MaxMessageLength {
		return ErrMessageTooLong
	}
	return nil
}

// WSMessage — только для push-уведомлений
type WSMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
