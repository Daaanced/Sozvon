// Authorization/auth/auth.go
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token expired")
	ErrWeakPassword     = errors.New("password too weak")
	ErrPasswordMismatch = errors.New("password mismatch")
)

const (
	// Стоимость хеширования bcrypt (10-14 рекомендуется)
	bcryptCost     = 10
	minPasswordLen = 2
	maxPasswordLen = 72 // Ограничение bcrypt
)

// PasswordHasher интерфейс для хеширования паролей
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// JWTService сервис для работы с JWT токенами
type JWTService struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTService создает новый сервис JWT
func NewJWTService(secretKey []byte, duration time.Duration) *JWTService {
	return &JWTService{
		secretKey:     secretKey,
		tokenDuration: duration,
	}
}

// Claims структура для JWT claims
type Claims struct {
	Login  string `json:"login"`
	UserID string `json:"user_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken создает JWT токен для пользователя
func (s *JWTService) GenerateToken(login string) (string, error) {
	if login == "" {
		return "", errors.New("login cannot be empty")
	}

	claims := &Claims{
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "sozvon-auth",
			Subject:   login,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
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

	return claims, nil
}

// BcryptHasher реализация хеширования через bcrypt
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создает новый hasher с заданной стоимостью
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{
		cost: bcryptCost,
	}
}

// Hash хеширует пароль
func (h *BcryptHasher) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(bytes), nil
}

// Compare проверяет соответствие пароля хешу
func (h *BcryptHasher) Compare(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return fmt.Errorf("password comparison failed: %w", err)
	}
	return nil
}

// validatePassword проверяет требования к паролю
func validatePassword(password string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("%w: minimum length is %d characters", ErrWeakPassword, minPasswordLen)
	}
	if len(password) > maxPasswordLen {
		return fmt.Errorf("%w: maximum length is %d characters", ErrWeakPassword, maxPasswordLen)
	}
	return nil
}
