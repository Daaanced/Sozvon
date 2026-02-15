// Gateway/auth/jwt.go
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrMissingClaim = errors.New("missing required claim")
)

// JWTValidator интерфейс для валидации JWT токенов
type JWTValidator interface {
	ValidateToken(tokenStr string) (*Claims, error)
}

// Claims содержит данные из JWT токена
type Claims struct {
	Login  string
	UserID string
	jwt.RegisteredClaims
}

// JWTService сервис для работы с JWT
type JWTService struct {
	secretKey []byte
}

// NewJWTService создает новый экземпляр JWTService
func NewJWTService(secretKey []byte) *JWTService {
	return &JWTService{
		secretKey: secretKey,
	}
}

// ValidateToken валидирует JWT токен и возвращает claims
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

	// Проверка обязательных полей
	if claims.Login == "" {
		return nil, fmt.Errorf("%w: login", ErrMissingClaim)
	}

	return claims, nil
}

// ValidateJWT - обратная совместимость со старым API
func ValidateJWT(tokenStr string, secretKey []byte) (string, error) {
	service := NewJWTService(secretKey)
	claims, err := service.ValidateToken(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.Login, nil
}
