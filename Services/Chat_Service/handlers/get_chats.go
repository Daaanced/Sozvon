package handlers

import (
	"encoding/json"
	"net/http"

	"Chat_Service/models"
	"Chat_Service/storage"
)

func GetChats(w http.ResponseWriter, r *http.Request) {
	login := r.Header.Get("Authorization")
	if login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	storage.InboxMu.Lock()
	chatIDs := storage.UserInboxes[login]
	storage.InboxMu.Unlock()

	var result []models.ChatListItem

	for _, chatID := range chatIDs {
		storage.Mu.Lock()
		chat, ok := storage.Chats[chatID]
		storage.Mu.Unlock()
		if !ok {
			continue
		}

		result = append(result, models.ChatListItem{
			ChatID:      chatID,
			Members:     chat.Members,
			LastMessage: "", // TODO: когда появится MessageStorage
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
