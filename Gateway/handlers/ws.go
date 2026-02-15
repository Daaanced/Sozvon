// Gateway/handlers/ws.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"Gateway/auth"
	"Gateway/config"

	"github.com/gorilla/websocket"
)

const (
	// Время ожидания записи в WebSocket
	writeWait = 10 * time.Second
	// Время ожидания pong сообщения от клиента
	pongWait = 60 * time.Second
	// Интервал отправки ping сообщений
	pingPeriod = (pongWait * 9) / 10
	// Максимальный размер сообщения
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client представляет WebSocket клиента
type Client struct {
	conn     *websocket.Conn
	chatConn *websocket.Conn
	login    string
	send     chan []byte
	ctx      context.Context
	cancel   context.CancelFunc
}

// WebSocketHandler управляет WebSocket подключениями
type WebSocketHandler struct {
	config     *config.Config
	jwtService *auth.JWTService
	clients    sync.Map // login -> *Client
}

// NewWebSocketHandler создает новый WebSocket handler
func NewWebSocketHandler(cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		config:     cfg,
		jwtService: auth.NewJWTService(cfg.JWT.SecretKey),
	}
}

// HandleWebSocket обрабатывает новые WebSocket подключения
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Валидация JWT токена
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		log.Printf("JWT validation error: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Проверка, не подключен ли пользователь уже
	if _, exists := h.clients.Load(claims.Login); exists {
		http.Error(w, "User already connected", http.StatusConflict)
		return
	}

	// Upgrade соединения до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Настройка WebSocket соединения
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Подключение к Chat Service
	chatConn, err := h.connectToChatService(claims.Login)
	if err != nil {
		log.Printf("Failed to connect to chat service: %v", err)
		conn.Close()
		return
	}

	// Создание клиента
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		conn:     conn,
		chatConn: chatConn,
		login:    claims.Login,
		send:     make(chan []byte, 256),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Сохранение клиента
	h.clients.Store(claims.Login, client)
	log.Printf("Client connected: %s", claims.Login)

	// Запуск горутин для обработки сообщений
	go client.readFromClient()
	go client.writeToClient()
	go client.readFromChatService()
	go client.writeToChatService()
	go client.pingClient()
}

// connectToChatService устанавливает WebSocket соединение с Chat Service
func (h *WebSocketHandler) connectToChatService(login string) (*websocket.Conn, error) {
	chatURL := url.URL{
		Scheme:   "ws",
		Host:     h.getChatServiceHost(),
		Path:     "/ws",
		RawQuery: fmt.Sprintf("login=%s", url.QueryEscape(login)),
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(chatURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial chat service: %w", err)
	}

	return conn, nil
}

// getChatServiceHost извлекает хост из URL Chat Service
func (h *WebSocketHandler) getChatServiceHost() string {
	u, err := url.Parse(h.config.Services.ChatServiceURL)
	if err != nil {
		return "localhost:8084"
	}
	return u.Host
}

// readFromClient читает сообщения от клиента и отправляет в Chat Service
func (c *Client) readFromClient() {
	defer func() {
		c.cleanup()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for %s: %v", c.login, err)
				}
				return
			}

			// Отправка сообщения в Chat Service
			if err := c.chatConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing to chat service for %s: %v", c.login, err)
				return
			}
		}
	}
}

// writeToClient отправляет сообщения клиенту
func (c *Client) writeToClient() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.cleanup()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing to client %s: %v", c.login, err)
				return
			}
		}
	}
}

// readFromChatService читает сообщения из Chat Service
func (c *Client) readFromChatService() {
	defer c.cleanup()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.chatConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Chat service read error for %s: %v", c.login, err)
				}
				return
			}

			// Отправка сообщения клиенту через канал
			select {
			case c.send <- message:
			case <-c.ctx.Done():
				return
			default:
				log.Printf("Send channel full for client %s, dropping message", c.login)
			}
		}
	}
}

// writeToChatService пока не используется, но может быть полезна для буферизации
func (c *Client) writeToChatService() {
	// Резерв для будущего функционала
}

// pingClient отправляет ping сообщения клиенту
func (c *Client) pingClient() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error for %s: %v", c.login, err)
				return
			}
		}
	}
}

// cleanup закрывает соединения и освобождает ресурсы
func (c *Client) cleanup() {
	c.cancel()

	if c.conn != nil {
		c.conn.Close()
	}

	if c.chatConn != nil {
		c.chatConn.Close()
	}

	close(c.send)

	log.Printf("Client disconnected: %s", c.login)
}

// DisconnectClient отключает клиента по логину (полезно для админских функций)
func (h *WebSocketHandler) DisconnectClient(login string) {
	if client, ok := h.clients.LoadAndDelete(login); ok {
		if c, ok := client.(*Client); ok {
			c.cleanup()
		}
	}
}
