// User_Service/models/user.go
package models

import (
	"errors"
	"regexp"
	"time"
)

var (
	ErrInvalidLogin = errors.New("invalid login format")
	ErrInvalidEmail = errors.New("invalid email format")
)

// User представляет профиль пользователя
type User struct {
	ID        int       `json:"id"`
	Login     string    `json:"login"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Info      string    `json:"info"`
	Picture   string    `json:"picture"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest запрос на создание пользователя
type CreateUserRequest struct {
	Login   string `json:"login" validate:"required,min=3,max=50"`
	Name    string `json:"name" validate:"required,min=1,max=100"`
	Email   string `json:"email" validate:"omitempty,email,max=255"`
	Info    string `json:"info" validate:"max=500"`
	Picture string `json:"picture" validate:"max=255"`
}

// UpdateUserRequest запрос на обновление пользователя
type UpdateUserRequest struct {
	Name    string `json:"name" validate:"omitempty,min=1,max=100"`
	Email   string `json:"email" validate:"omitempty,email,max=255"`
	Info    string `json:"info" validate:"max=500"`
	Picture string `json:"picture" validate:"max=255"`
}

// UserListResponse ответ со списком пользователей
type UserListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
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

// Validate валидирует CreateUserRequest
func (r *CreateUserRequest) Validate() error {
	if len(r.Login) < 3 || len(r.Login) > 50 {
		return ErrInvalidLogin
	}

	if r.Email != "" && !isValidEmail(r.Email) {
		return ErrInvalidEmail
	}

	if len(r.Name) < 1 || len(r.Name) > 100 {
		return errors.New("name must be between 1 and 100 characters")
	}

	if len(r.Info) > 500 {
		return errors.New("info must be less than 500 characters")
	}

	return nil
}

// Validate валидирует UpdateUserRequest
func (r *UpdateUserRequest) Validate() error {
	if r.Name != "" && (len(r.Name) < 1 || len(r.Name) > 100) {
		return errors.New("name must be between 1 and 100 characters")
	}

	if r.Email != "" && !isValidEmail(r.Email) {
		return ErrInvalidEmail
	}

	if len(r.Info) > 500 {
		return errors.New("info must be less than 500 characters")
	}

	return nil
}

// isValidEmail проверяет корректность email
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ToUser конвертирует CreateUserRequest в User
func (r *CreateUserRequest) ToUser() User {
	return User{
		Login:   r.Login,
		Name:    r.Name,
		Email:   r.Email,
		Info:    r.Info,
		Picture: r.Picture,
	}
}
