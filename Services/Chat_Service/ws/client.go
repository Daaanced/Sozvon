// Chat_Service/ws/client.go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"Chat_Service/models"

	"github.com/gorilla/websocket"
)

// Client представляет WebSocket клиента
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Login  string
	send   chan models.WSMessage
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient создает нового WebSocket клиента
func NewClient(hub *Hub, conn *websocket.Conn, login string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		Hub:    hub,
		Conn:   conn,
		Login:  login,
		send:   make(chan models.WSMessage, 256),
		ctx:    ctx,
		cancel: cancel,
	}

	// Настройка WebSocket соединения
	conn.SetReadLimit(hub.config.WebSocket.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(hub.config.WebSocket.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(hub.config.WebSocket.PongWait))
		return nil
	})

	return client
}

// Start запускает горутины для чтения и записи
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump читает сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var msg models.WSMessage
			err := c.Conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for %s: %v", c.Login, err)
				}
				return
			}

			// Обработка сообщения через Hub
			c.Hub.HandleMessage(c, msg)
		}
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(c.Hub.config.WebSocket.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return

		case message, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.config.WebSocket.WriteWait))
			if !ok {
				// Канал закрыт
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error for %s: %v", c.Login, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.config.WebSocket.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error for %s: %v", c.Login, err)
				return
			}
		}
	}
}

// SendMessage отправляет сообщение клиенту
func (c *Client) SendMessage(message models.WSMessage) {
	select {
	case c.send <- message:
	case <-c.ctx.Done():
	default:
		log.Printf("Warning: send buffer full for %s", c.Login)
	}
}

// SendError отправляет сообщение об ошибке клиенту
func (c *Client) SendError(code, message string) {
	errorMsg := models.WSMessage{
		Event: "error",
		Data: models.ErrorResponse{
			Error:   code,
			Message: message,
		},
	}
	c.SendMessage(errorMsg)
}

// Close закрывает соединение клиента
func (c *Client) Close() {
	c.cancel()

	if c.Conn != nil {
		c.Conn.Close()
	}

	// Закрытие канала send (если еще не закрыт)
	select {
	case <-c.send:
		// Уже закрыт
	default:
		close(c.send)
	}
}

// MarshalJSON для сериализации клиента (для отладки)
func (c *Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Login string `json:"login"`
	}{
		Login: c.Login,
	})
}
