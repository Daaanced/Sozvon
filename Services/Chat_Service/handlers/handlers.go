// Chat_Service/handlers/handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Chat_Service/auth"
	"Chat_Service/config"
	"Chat_Service/db"
	"Chat_Service/models"
	"Chat_Service/ws"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ChatHandler struct {
	config     *config.Config
	db         *db.Database
	hub        *ws.Hub
	jwtService *auth.JWTService
}

func NewChatHandler(cfg *config.Config, database *db.Database, hub *ws.Hub) *ChatHandler {
	return &ChatHandler{
		config:     cfg,
		db:         database,
		hub:        hub,
		jwtService: auth.NewJWTService(cfg.JWT.SecretKey),
	}
}

func (h *ChatHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws", h.HandleWebSocket)
	r.HandleFunc("/chats/create", h.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats", h.GetChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages", h.GetMessages).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages", h.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats/{chatId}", h.GetChatInfo).Methods("GET", "OPTIONS")
	r.HandleFunc("/internal/members/{userId}", h.DeleteMembersByUserID).Methods("DELETE")
}

// HandleWebSocket — теперь идентифицируем по UserID из JWT
func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.UserID)
	if err != nil || userID == 0 {
		http.Error(w, "invalid user_id in token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, userID)
	h.hub.Register <- client
	client.Start()
}

// CreateChat — принимает from_id / to_id
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	// from_id — из токена, не из тела
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

	chatID, err := h.db.CreateChat(ctx, []int{fromID, body.ToID}, true)
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

// SendMessage — POST /chats/{chatId}/messages
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Проверка членства
	isMember, err := h.db.IsMember(ctx, chatID, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to check membership")
		return
	}
	if !isMember {
		respondWithError(w, http.StatusForbidden, "forbidden", "You are not a member of this chat")
		return
	}

	// Сохранение
	msg, err := h.db.SaveMessage(ctx, chatID, userID, req.Text)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to save message")
		return
	}

	// Активация чата при первом сообщении
	if activated, _ := h.db.ActivateChat(ctx, chatID); activated {
		members, _ := h.db.GetChatMembers(ctx, chatID)
		h.hub.SendToUsers(members, models.WSMessage{
			Event: "chat:activated",
			Data:  map[string]string{"chatId": chatID},
		})
	}

	// Push всем участникам кроме отправителя
	members, err := h.db.GetChatMembers(ctx, chatID)
	if err == nil {
		recipients := make([]int, 0, len(members)-1)
		for _, uid := range members {
			if uid != userID {
				recipients = append(recipients, uid)
			}
		}
		h.hub.SendToUsers(recipients, models.WSMessage{
			Event: "message:new",
			Data:  msg,
		})
	}

	respondWithJSON(w, http.StatusCreated, msg)
}

// GetChats — требует Authorization header
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

// GetMessages — GET /chats/{chatId}/messages
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	if chatID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Chat ID required")
		return
	}

	limit, offset := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	exists, err := h.db.ChatExists(ctx, chatID)
	if err != nil || !exists {
		respondWithError(w, http.StatusNotFound, "chat_not_found", "Chat not found")
		return
	}

	messages, err := h.db.GetChatMessages(ctx, chatID, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get messages")
		return
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// GetChatInfo — участники и онлайн-статус
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

// DeleteMembersByUserID — internal endpoint
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

// --- helpers ---

func (h *ChatHandler) extractUserIDFromAuth(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, fmt.Errorf("missing or invalid authorization header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	userID, err := strconv.Atoi(claims.UserID)
	if err != nil || userID == 0 {
		return 0, fmt.Errorf("invalid user_id in token")
	}

	return userID, nil
}

func (h *ChatHandler) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	return
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, code, message string) {
	respondWithJSON(w, statusCode, models.ErrorResponse{Error: code, Message: message})
}
