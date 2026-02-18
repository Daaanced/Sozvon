// Gateway/handlers/ws.go - ИСПРАВЛЕННАЯ ВЕРСИЯ
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
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	conn     *websocket.Conn
	chatConn *websocket.Conn
	login    string
	token    string
	send     chan []byte
	ctx      context.Context
	cancel   context.CancelFunc
}

type WebSocketHandler struct {
	config     *config.Config
	jwtService *auth.JWTService
	clients    sync.Map
}

func NewWebSocketHandler(cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		config:     cfg,
		jwtService: auth.NewJWTService(cfg.JWT.SecretKey),
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
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

	// ✅ ИСПРАВЛЕНИЕ: Закрываем старое соединение вместо отклонения нового
	if existing, exists := h.clients.Load(claims.Login); exists {
		if oldClient, ok := existing.(*Client); ok {
			log.Printf("Closing old connection for %s (user refreshed page)", claims.Login)
			oldClient.cleanup()
			h.clients.Delete(claims.Login)
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	chatConn, err := h.connectToChatService(token)
	if err != nil {
		log.Printf("Failed to connect to chat service: %v", err)
		conn.Close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		conn:     conn,
		chatConn: chatConn,
		login:    claims.Login,
		token:    token,
		send:     make(chan []byte, 256),
		ctx:      ctx,
		cancel:   cancel,
	}

	h.clients.Store(claims.Login, client)
	log.Printf("Client connected: %s", claims.Login)

	go client.readFromClient()
	go client.writeToClient()
	go client.readFromChatService()
	go client.pingClient()
}

func (h *WebSocketHandler) connectToChatService(token string) (*websocket.Conn, error) {
	chatURL := url.URL{
		Scheme:   "ws",
		Host:     h.getChatServiceHost(),
		Path:     "/ws",
		RawQuery: fmt.Sprintf("token=%s", url.QueryEscape(token)),
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

func (h *WebSocketHandler) getChatServiceHost() string {
	u, err := url.Parse(h.config.Services.ChatServiceURL)
	if err != nil {
		return "localhost:8084"
	}
	return u.Host
}

func (c *Client) readFromClient() {
	defer c.cleanup()

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

			if err := c.chatConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing to chat service for %s: %v", c.login, err)
				return
			}
		}
	}
}

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

func (c *Client) cleanup() {
	c.cancel()

	if c.conn != nil {
		c.conn.Close()
	}

	if c.chatConn != nil {
		c.chatConn.Close()
	}

	// ✅ Безопасное закрытие канала
	select {
	case <-c.send:
		// Уже закрыт
	default:
		close(c.send)
	}

	log.Printf("Client disconnected: %s", c.login)
}

func (h *WebSocketHandler) DisconnectClient(login string) {
	if client, ok := h.clients.LoadAndDelete(login); ok {
		if c, ok := client.(*Client); ok {
			c.cleanup()
		}
	}
}
