package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"Chat_Service/db"
)

type Message struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/messages") {
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "bad request", 400)
		return
	}

	chatID := parts[2]

	rows, err := db.DB.Query(`
		SELECT id, sender_login, text, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
	`, chatID)

	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer rows.Close()

	var result []Message

	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.From, &m.Text, &m.CreatedAt); err != nil {
			continue
		}
		result = append(result, m)
	}

	json.NewEncoder(w).Encode(result)
}
