// Chat_Service\handlers\chats.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"Chat_Service/models"

	"github.com/gorilla/mux"
)

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	fromID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var body struct {
		ToID int `json:"to_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ToID == 0 {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "to_id required")
		return
	}
	if fromID == body.ToID {
		respondWithError(w, http.StatusBadRequest, "validation_error", "cannot create chat with yourself")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	existingID, err := h.db.FindExistingChat(ctx, fromID, body.ToID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to find chat")
		return
	}
	if existingID != "" {
		respondWithJSON(w, http.StatusOK, models.Chat{
			ID:      existingID,
			Members: []int{fromID, body.ToID},
			Active:  true,
		})
		return
	}

	chatID, err := h.db.CreateChat(ctx, []int{fromID, body.ToID}, true, "direct", "")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to create chat")
		return
	}

	h.hub.SendToUsers([]int{fromID, body.ToID}, models.WSMessage{
		Event: "chat:created",
		Data:  map[string]string{"chatId": chatID},
	})

	log.Printf("Chat created: %s [%d, %d]", chatID, fromID, body.ToID)
	respondWithJSON(w, http.StatusCreated, models.Chat{
		ID:      chatID,
		Members: []int{fromID, body.ToID},
		Active:  true,
	})
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	chats, err := h.db.GetUserChats(ctx, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get chats")
		return
	}

	respondWithJSON(w, http.StatusOK, chats)
}

func (h *ChatHandler) GetChatInfo(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	members, err := h.db.GetChatMembers(ctx, chatID)
	if err != nil || len(members) == 0 {
		respondWithError(w, http.StatusNotFound, "chat_not_found", "Chat not found")
		return
	}

	onlineStatus := make(map[int]bool, len(members))
	for _, uid := range members {
		onlineStatus[uid] = h.hub.IsUserOnline(uid)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"chatId":       chatID,
		"members":      members,
		"onlineStatus": onlineStatus,
	})
}

func (h *ChatHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var body struct {
		LastMessageID string `json:"lastMessageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LastMessageID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "lastMessageId required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.MarkChatRead(ctx, chatID, userID, body.LastMessageID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to mark read")
		return
	}

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{Status: "ok"})
}

func (h *ChatHandler) DeleteMembersByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := mux.Vars(r)["userId"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID == 0 {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Valid user ID required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.DeleteChatMembersByUserID(ctx, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to delete members")
		return
	}

	log.Printf("Deleted %d chat_members for userID=%d", rows, userID)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"userId":       userID,
		"rowsAffected": rows,
	})
}
