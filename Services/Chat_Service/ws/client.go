// Chat_Service/ws/client.go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"Chat_Service/models"

	"github.com/gorilla/websocket"
)

type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	UserID    int // вместо Login
	send      chan models.WSMessage
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func NewClient(hub *Hub, conn *websocket.Conn, userID int) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		Hub:    hub,
		Conn:   conn,
		UserID: userID,
		send:   make(chan models.WSMessage, 256),
		ctx:    ctx,
		cancel: cancel,
	}

	conn.SetReadLimit(hub.config.WebSocket.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(hub.config.WebSocket.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(hub.config.WebSocket.PongWait))
		return nil
	})

	return client
}

func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

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
			if err := c.Conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WS read error userID=%d: %v", c.UserID, err)
				}
				return
			}
			c.Hub.HandleMessage(c, msg)
		}
	}
}

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
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WS write error userID=%d: %v", c.UserID, err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.config.WebSocket.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendMessage(message models.WSMessage) {
	select {
	case c.send <- message:
	case <-c.ctx.Done():
	default:
		log.Printf("Warning: send buffer full userID=%d", c.UserID)
	}
}

func (c *Client) SendError(code, message string) {
	c.SendMessage(models.WSMessage{
		Event: "error",
		Data:  models.ErrorResponse{Error: code, Message: message},
	})
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()
		if c.Conn != nil {
			c.Conn.Close()
		}
		close(c.send)
		log.Printf("WS connection closed userID=%d", c.UserID)
	})
}

func (c *Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		UserID int `json:"userId"`
	}{UserID: c.UserID})
}
