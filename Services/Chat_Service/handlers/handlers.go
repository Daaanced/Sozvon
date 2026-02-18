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

// ChatHandler обрабатывает запросы чатов
type ChatHandler struct {
	config     *config.Config
	db         *db.Database
	hub        *ws.Hub
	jwtService *auth.JWTService
}

// NewChatHandler создает новый обработчик чатов
func NewChatHandler(cfg *config.Config, database *db.Database, hub *ws.Hub) *ChatHandler {
	return &ChatHandler{
		config:     cfg,
		db:         database,
		hub:        hub,
		jwtService: auth.NewJWTService(cfg.JWT.SecretKey),
	}
}

// RegisterRoutes регистрирует маршруты
func (h *ChatHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws", h.HandleWebSocket)
	r.HandleFunc("/chats/create", h.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/chats", h.GetChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}/messages", h.GetMessages).Methods("GET", "OPTIONS")
	r.HandleFunc("/chats/{chatId}", h.GetChatInfo).Methods("GET", "OPTIONS")
}

// HandleWebSocket обрабатывает WebSocket подключения
func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	var login string

	// Получаем token из query параметра
	token := r.URL.Query().Get("token")
	if token != "" {
		// Валидируем JWT
		claims, err := h.jwtService.ValidateToken(token)
		if err != nil {
			log.Printf("JWT validation error: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		login = claims.Login
	}

	// Upgrade до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(h.config.WebSocket.MaxMessageSize)

	// Создание клиента
	client := ws.NewClient(h.hub, conn, login)

	// Регистрация в hub
	h.hub.Register <- client

	// Запуск обработки
	client.Start()
}

// CreateChat создает новый чат
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var req models.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
		return
	}

	// Валидация
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Проверка существующего чата
	existingChatID, err := h.db.FindExistingChat(ctx, req.From, req.To)
	if err != nil {
		log.Printf("Error finding chat: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to find chat")
		return
	}

	var chatID string
	var isNew bool

	if existingChatID != "" {
		chatID = existingChatID
		isNew = false
	} else {
		// Создание нового чата
		chatID, err = h.db.CreateChat(ctx, []string{req.From, req.To}, true)
		if err != nil {
			log.Printf("Error creating chat: %v", err)
			respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to create chat")
			return
		}
		isNew = true

		// Уведомление участников через WebSocket
		notification := models.WSMessage{
			Event: "chat:created",
			Data: map[string]string{
				"chatId": chatID,
			},
		}
		h.hub.SendToUsers([]string{req.From, req.To}, notification)
	}

	respondWithJSON(w, http.StatusOK, models.Chat{
		ID:      chatID,
		Members: []string{req.From, req.To},
		Active:  true,
	})

	if isNew {
		log.Printf("Chat created: %s [%s, %s]", chatID, req.From, req.To)
	}
}

// GetChats возвращает список чатов пользователя
func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	// Извлечение login из JWT
	login, err := h.extractLoginFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	chats, err := h.db.GetUserChats(ctx, login)
	if err != nil {
		log.Printf("Error getting chats: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get chats")
		return
	}

	respondWithJSON(w, http.StatusOK, chats)
}

// GetMessages возвращает сообщения чата
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chatId"]

	if chatID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Chat ID required")
		return
	}

	// Пагинация
	limit, offset := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Проверка существования чата
	exists, err := h.db.ChatExists(ctx, chatID)
	if err != nil {
		log.Printf("Error checking chat existence: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to check chat")
		return
	}

	if !exists {
		respondWithError(w, http.StatusNotFound, "chat_not_found", "Chat not found")
		return
	}

	// Получение сообщений
	messages, err := h.db.GetChatMessages(ctx, chatID, limit, offset)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get messages")
		return
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// GetChatInfo возвращает информацию о чате
func (h *ChatHandler) GetChatInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chatId"]

	if chatID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Chat ID required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Получение участников
	members, err := h.db.GetChatMembers(ctx, chatID)
	if err != nil {
		log.Printf("Error getting chat info: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get chat info")
		return
	}

	if len(members) == 0 {
		respondWithError(w, http.StatusNotFound, "chat_not_found", "Chat not found")
		return
	}

	// Проверка онлайн статуса участников
	onlineStatus := make(map[string]bool)
	for _, member := range members {
		onlineStatus[member] = h.hub.IsUserOnline(member)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"chatId":       chatID,
		"members":      members,
		"onlineStatus": onlineStatus,
	})
}

// --- Вспомогательные функции ---

// extractLoginFromAuth извлекает login из Authorization header
func (h *ChatHandler) extractLoginFromAuth(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("missing or invalid authorization header")
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return claims.Login, nil
}

// getPaginationParams извлекает параметры пагинации
func (h *ChatHandler) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 50 // По умолчанию
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, code, message string) {
	respondWithJSON(w, statusCode, models.ErrorResponse{
		Error:   code,
		Message: message,
	})
}
