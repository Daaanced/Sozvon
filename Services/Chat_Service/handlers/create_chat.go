// Chat_Service\handlers\create_chat.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"Chat_Service/db"
	"Chat_Service/models"
)

type CreateChatRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	if req.From == "" || req.To == "" {
		http.Error(w, "from and to required", 400)
		return
	}

	var chatID string

	// 1️⃣ ищем существующий чат
	err := db.DB.QueryRow(`
		SELECT c.id
		FROM chats c
		JOIN chat_members m1 ON m1.chat_id = c.id AND m1.login = $1
		JOIN chat_members m2 ON m2.chat_id = c.id AND m2.login = $2
		LIMIT 1
	`, req.From, req.To).Scan(&chatID)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "db error", 500)
		return
	}

	// 2️⃣ если нет — создаём
	if err == sql.ErrNoRows {
		chatID = uuid.NewString()

		tx, err := db.DB.Begin()
		if err != nil {
			http.Error(w, "tx error", 500)
			return
		}

		_, err = tx.Exec(`INSERT INTO chats (id, active) VALUES ($1, true)`, chatID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "insert chat error", 500)
			return
		}

		_, err = tx.Exec(`
			INSERT INTO chat_members (chat_id, login)
			VALUES ($1, $2), ($1, $3)
		`, chatID, req.From, req.To)
		if err != nil {
			tx.Rollback()
			http.Error(w, "insert members error", 500)
			return
		}

		tx.Commit()
		log.Printf("chat created: %s [%s %s]", chatID, req.From, req.To)
	}

	json.NewEncoder(w).Encode(models.Chat{
		ID:      chatID,
		Members: []string{req.From, req.To},
		Active:  true, // логически активен
	})
}
