// Gateway\auth\jwt.go
package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("supersecretkey") // ⚠️ потом вынести в env

func ValidateJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	login := claims["login"].(string)

	return login, nil
}
