// voice_service/auth/jwt.go
//
// Тот же JWT секрет что у Gateway и Auth Service.
// Voice Service валидирует токен самостоятельно — не обращается к Auth Service,
// чтобы не добавлять зависимость и latency на критическом пути сигнализации.

package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Login    string `json:"login"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// Fallback: если UserID не заполнен — используем Login
	// (зависит от того что кладёт Auth Service в токен)
	if claims.UserID == "" {
		claims.UserID = claims.Login
	}
	if claims.Username == "" {
		claims.Username = claims.Login
	}

	return claims, nil
}
