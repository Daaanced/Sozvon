// Chat_Service\handlers\get_chats.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"

	"Chat_Service/db"
	"Chat_Service/models"
)

func GetChats(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "unauthorized", 401)
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	login, err := ValidateJWT(token)

	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	log.Println("GetChats login:", login)

	rows, err := db.DB.Query(`
		SELECT
			c.id,
			array_agg(cm.login) AS members,
			COALESCE(m.text, '') AS last_message
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id
		LEFT JOIN LATERAL (
			SELECT text
			FROM messages
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		WHERE c.id IN (
			SELECT chat_id FROM chat_members WHERE login = $1
		)
		GROUP BY c.id, m.text
		ORDER BY MAX(c.created_at) DESC
	`, login)

	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer rows.Close()

	result := make([]models.ChatListItem, 0)

	for rows.Next() {
		var item models.ChatListItem
		var members []string

		if err := rows.Scan(
			&item.ChatID,
			pq.Array(&members),
			&item.LastMessage,
		); err != nil {
			log.Println("scan error:", err)
			continue
		}

		item.Members = members
		result = append(result, item)
	}

	json.NewEncoder(w).Encode(result)
}

func ValidateJWT(tokenStr string) (string, error) {
	var jwtKey = []byte("supersecretkey") // ⚠️ потом вынести в env

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	login := claims["login"].(string)
	log.Println("JWT claims:", claims)
	return login, nil
}
