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
	Members   []int     `json:"members"`
	Active    bool      `json:"active"`
	Type      string    `json:"type"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatListItem struct {
	ChatID      string    `json:"chatId"`
	Type        string    `json:"type"`
	Name        string    `json:"name,omitempty"`
	Members     []int     `json:"members"`
	LastMessage string    `json:"lastMessage"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Attachment — вложение к сообщению
type Attachment struct {
	ID        string `json:"id"`
	MessageID string `json:"messageId"`
	FileName  string `json:"fileName"`
	StoreName string `json:"-"` // имя файла на диске, не отдаём клиенту
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	URL       string `json:"url"` // заполняется при отдаче, не хранится в БД
}

type Message struct {
	ID              string        `json:"id"`
	ChatID          string        `json:"chatId"`
	SenderID        int           `json:"senderId"`
	Text            string        `json:"text"`
	ReplyToID       *string       `json:"replyToId,omitempty"`
	ReplyToMessage  *ReplyPreview `json:"replyToMessage,omitempty"`
	ForwardedFromID *string       `json:"forwardedFromId,omitempty"`
	Attachments     []Attachment  `json:"attachments,omitempty"`
	CreatedAt       time.Time     `json:"createdAt"`
}

type ReplyPreview struct {
	ID       string `json:"id"`
	SenderID int    `json:"senderId"`
	Text     string `json:"text"`
}

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

type SendMessageRequest struct {
	Text            string  `json:"text"`
	ReplyToID       *string `json:"replyToId,omitempty"`
	ForwardedFromID *string `json:"forwardedFromId,omitempty"`
}

func (r *SendMessageRequest) Validate() error {
	if r.ForwardedFromID == nil && r.Text == "" {
		return ErrEmptyMessage
	}
	if len(r.Text) > MaxMessageLength {
		return ErrMessageTooLong
	}
	return nil
}

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
