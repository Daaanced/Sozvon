// Chat_Service\handlers\create_chat.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"Chat_Service/models"
	"Chat_Service/storage"
)

type CreateChatRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.From == "" || req.To == "" {
		http.Error(w, "from and to required", http.StatusBadRequest)
		return
	}

	chatID := uuid.NewString()

	chat := models.Chat{
		ID:      chatID,
		Members: []string{req.From, req.To},
	}

	chatID, exists := FindChat(req.From, req.To)
	if !exists {
		chatID = uuid.NewString()
		storage.Mu.Lock()
		storage.Chats[chatID] = models.Chat{
			ID:      chatID,
			Members: []string{req.From, req.To},
		}
		storage.Mu.Unlock()
		log.Printf("chat created: id=%s members=[%s %s]", chatID, req.From, req.To)
	}

	// Формируем ответ
	chat = storage.Chats[chatID]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)

}

func FindChat(userA, userB string) (string, bool) {
	storage.Mu.Lock()
	defer storage.Mu.Unlock()

	for id, chat := range storage.Chats {
		if contains(chat.Members, userA) && contains(chat.Members, userB) {
			return id, true
		}
	}
	return "", false
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
