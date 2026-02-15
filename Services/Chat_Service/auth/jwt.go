// Chat_Service/auth/jwt.go
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// JWTService сервис для работы с JWT
type JWTService struct {
	secretKey []byte
}

// NewJWTService создает новый сервис JWT
func NewJWTService(secretKey []byte) *JWTService {
	return &JWTService{
		secretKey: secretKey,
	}
}

// Claims структура для JWT claims
type Claims struct {
	Login  string `json:"login"`
	UserID string `json:"user_id,omitempty"`
	jwt.RegisteredClaims
}

// ValidateToken валидирует JWT токен
func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверка алгоритма подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
