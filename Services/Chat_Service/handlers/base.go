// Chat_Service\handlers\base.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"Chat_Service/auth"
	"Chat_Service/config"
	"Chat_Service/db"
	"Chat_Service/models"
	"Chat_Service/storage"
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
	storage    *storage.FileStorage
}

func NewChatHandler(cfg *config.Config, database *db.Database, hub *ws.Hub) *ChatHandler {
	fileStorage, err := storage.NewFileStorage(cfg.Media.Directory, cfg.Media.BaseURL)
	if err != nil {
		log.Fatalf("Failed to init file storage: %v", err)
	}
	return &ChatHandler{
		config:     cfg,
		db:         database,
		hub:        hub,
		jwtService: auth.NewJWTService(cfg.JWT.SecretKey),
		storage:    fileStorage,
	}
}

func (h *ChatHandler) RegisterRoutes(r *mux.Router) {
	// WebSocket
	r.HandleFunc("/ws", h.HandleWebSocket)

	// Чаты
	r.HandleFunc("/chats/create", h.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats/forward", h.ForwardMessages).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats", h.GetChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}", h.GetChatInfo).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/read", h.MarkRead).Methods("POST", "OPTIONS")

	// Сообщения
	r.HandleFunc("/chats/{chatId}/messages", h.GetMessages).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages", h.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages/{messageId}/context", h.GetMessagesContext).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages/{messageId}", h.EditMessage).Methods("PUT", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages/{messageId}", h.DeleteMessage).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/upload", h.UploadFiles).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages/unread", h.GetUnreadMessages).Methods("GET", "OPTIONS")

	// Медиа
	r.HandleFunc("/media/{filename}", h.ServeMedia).Methods("GET")

	// Internal
	r.HandleFunc("/internal/members/{userId}", h.DeleteMembersByUserID).Methods("DELETE")
}

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
