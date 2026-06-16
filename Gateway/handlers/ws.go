// Gateway/handlers/ws.go
// ИЗМЕНЕНИЯ: добавлен voiceConn в Client, поддержка /ws/voice endpoint
// и маршрутизация сообщений по типу (chat vs voice)

package handlers

import (
	"context"
	"encoding/json"
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
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// targetService — куда маршрутизировать WS сообщение
type targetService int

const (
	serviceChatDefault targetService = iota
	serviceVoice
)

// routeMessage — определяет сервис по типу сообщения.
// Голосовые сообщения имеют поле "service": "voice" или известные типы сигнализации.
func routeMessage(data []byte) targetService {
	var envelope struct {
		Service string `json:"service"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return serviceChatDefault
	}

	if envelope.Service == "voice" {
		return serviceVoice
	}

	// Известные типы сигнализации WebRTC
	switch envelope.Type {
	case "join", "leave", "offer", "answer", "ice_candidate", "mute", "deafened", "set_layer":
		return serviceVoice
	}

	return serviceChatDefault
}

// Client — WS сессия одного пользователя на Gateway
type Client struct {
	conn      *websocket.Conn
	chatConn  *websocket.Conn
	voiceConn *websocket.Conn // nil если voice service не подключён
	login     string
	token     string
	send      chan []byte
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex // защищает запись в conn
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

// HandleWebSocket — основной WS endpoint (/ws).
// Подключает клиента к Chat Service автоматически.
// Voice Service подключается лениво при первом голосовом сообщении.
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

	// Закрываем старое соединение
	if existing, exists := h.clients.Load(claims.Login); exists {
		if oldClient, ok := existing.(*Client); ok {
			log.Printf("Closing old connection for %s", claims.Login)
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

	go client.readFromClient(h)
	go client.writeToClient()
	go client.readFromChatService()
	go client.pingClient()
}

// HandleVoiceWebSocket — отдельный endpoint /ws/voice.
// Подключает клиента только к Voice Service (без Chat Service).
// Используется если клиент хочет явно открыть отдельное соединение для голоса.
//
// В вашей текущей архитектуре это альтернативный вариант —
// можно использовать либо его, либо маршрутизацию через /ws.
func (h *WebSocketHandler) HandleVoiceWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	if _, err := h.jwtService.ValidateToken(token); err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Voice WS upgrade error: %v", err)
		return
	}

	voiceConn, err := h.connectToVoiceService(token)
	if err != nil {
		log.Printf("Failed to connect to voice service: %v", err)
		conn.Close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Простой прокси: клиент ↔ voice service
	go func() {
		defer cancel()
		defer conn.Close()
		defer voiceConn.Close()

		// клиент → voice
		go func() {
			for {
				mt, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if err := voiceConn.WriteMessage(mt, data); err != nil {
					return
				}
			}
		}()

		// voice → клиент
		for {
			select {
			case <-ctx.Done():
				return
			default:
				mt, data, err := voiceConn.ReadMessage()
				if err != nil {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(mt, data); err != nil {
					return
				}
			}
		}
	}()
}

// ── Client methods ─────────────────────────────────────────────────────────

// readFromClient — читает от клиента и маршрутизирует в нужный сервис
func (c *Client) readFromClient(h *WebSocketHandler) {
	defer c.cleanup()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
				) {
					log.Printf("WS read error for %s: %v", c.login, err)
				}
				return
			}

			target := routeMessage(message)

			switch target {
			case serviceVoice:
				// Лениво подключаемся к voice service
				if c.voiceConn == nil {
					vc, err := h.connectToVoiceService(c.token)
					if err != nil {
						log.Printf("Failed to connect to voice service for %s: %v", c.login, err)
						continue
					}
					c.voiceConn = vc
					// Запускаем reader для ответов от voice service
					go c.readFromVoiceService()
				}
				if err := c.voiceConn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("Error writing to voice service for %s: %v", c.login, err)
					c.voiceConn.Close()
					c.voiceConn = nil
				}

			default: // chat
				if err := c.chatConn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("Error writing to chat service for %s: %v", c.login, err)
					return
				}
			}
		}
	}
}

func (c *Client) writeToClient() {
	defer c.cleanup()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			c.mu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()
			if err != nil {
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
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
				) {
					log.Printf("Chat service read error for %s: %v", c.login, err)
				}
				return
			}
			select {
			case c.send <- message:
			case <-c.ctx.Done():
				return
			default:
				log.Printf("Send channel full for %s, dropping chat message", c.login)
			}
		}
	}
}

func (c *Client) readFromVoiceService() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.voiceConn == nil {
				return
			}
			_, message, err := c.voiceConn.ReadMessage()
			if err != nil {
				log.Printf("Voice service read error for %s: %v", c.login, err)
				return
			}
			select {
			case c.send <- message:
			case <-c.ctx.Done():
				return
			default:
				log.Printf("Send channel full for %s, dropping voice message", c.login)
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
			c.mu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()
			if err != nil {
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
	if c.voiceConn != nil {
		c.voiceConn.Close()
	}

	select {
	case <-c.send:
	default:
		close(c.send)
	}

	log.Printf("Client disconnected: %s", c.login)
}

// ── Service connections ────────────────────────────────────────────────────

func (h *WebSocketHandler) connectToChatService(token string) (*websocket.Conn, error) {
	u := url.URL{
		Scheme:   "ws",
		Host:     h.getServiceHost(h.config.Services.ChatServiceURL),
		Path:     "/ws",
		RawQuery: fmt.Sprintf("token=%s", url.QueryEscape(token)),
	}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial chat service: %w", err)
	}
	return conn, nil
}

func (h *WebSocketHandler) connectToVoiceService(token string) (*websocket.Conn, error) {
	u := url.URL{
		Scheme:   "ws",
		Host:     h.getServiceHost(h.config.Services.VoiceServiceURL),
		Path:     "/ws",
		RawQuery: fmt.Sprintf("token=%s", url.QueryEscape(token)),
	}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial voice service: %w", err)
	}
	return conn, nil
}

func (h *WebSocketHandler) getServiceHost(serviceURL string) string {
	u, err := url.Parse(serviceURL)
	if err != nil {
		return "localhost:8085"
	}
	return u.Host
}

func (h *WebSocketHandler) DisconnectClient(login string) {
	if client, ok := h.clients.LoadAndDelete(login); ok {
		if c, ok := client.(*Client); ok {
			c.cleanup()
		}
	}
}
