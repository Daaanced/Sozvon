// Authorization/models/models.go
package models

import "time"

// User представляет пользователя в системе аутентификации
type User struct {
	ID        int       `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"password,omitempty"` // Хеш пароля, не отдаем в JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest структура для запроса логина
type LoginRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// RegisterRequest структура для запроса регистрации
type RegisterRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// AuthResponse структура ответа при успешной аутентификации
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // Секунды до истечения
	TokenType string `json:"token_type"`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse общая структура успешного ответа
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
