// Chat_Service/ws/hub.go
package ws

import (
	"log"
	"sync"

	"Chat_Service/config"
	"Chat_Service/db"
	"Chat_Service/models"
)

type Hub struct {
	config     *config.Config
	db         *db.Database
	clients    sync.Map // userID (int) -> *Client
	Register   chan *Client
	Unregister chan *Client
	broadcast  chan *BroadcastMessage
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

type BroadcastMessage struct {
	Recipients []int // user IDs
	Message    models.WSMessage
}

func NewHub(cfg *config.Config, database *db.Database) *Hub {
	return &Hub{
		config:     cfg,
		db:         database,
		Register:   make(chan *Client, 256),
		Unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMessage, 1024),
		shutdown:   make(chan struct{}),
	}
}

func (h *Hub) Run() {
	h.wg.Add(1)
	defer h.wg.Done()

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.Unregister:
			h.unregisterClient(client)
		case msg := <-h.broadcast:
			h.broadcastMessage(msg)
		case <-h.shutdown:
			h.shutdownAllClients()
			return
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	if existing, exists := h.clients.LoadAndDelete(client.UserID); exists {
		if c, ok := existing.(*Client); ok {
			c.Close()
		}
	}
	h.clients.Store(client.UserID, client)
	log.Printf("WS client registered: userID=%d", client.UserID)
}

func (h *Hub) unregisterClient(client *Client) {
	h.clients.LoadAndDelete(client.UserID)
	log.Printf("WS client unregistered: userID=%d", client.UserID)
}

func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	for _, uid := range msg.Recipients {
		if v, ok := h.clients.Load(uid); ok {
			if c, ok := v.(*Client); ok {
				select {
				case c.send <- msg.Message:
				default:
					log.Printf("Warning: send buffer full for userID=%d", uid)
				}
			}
		}
	}
}

func (h *Hub) SendToUsers(userIDs []int, message models.WSMessage) {
	h.broadcast <- &BroadcastMessage{
		Recipients: userIDs,
		Message:    message,
	}
}

func (h *Hub) SendToUser(userID int, message models.WSMessage) {
	h.SendToUsers([]int{userID}, message)
}

// HandleMessage — WS принимает только ping/typing, не сообщения
func (h *Hub) HandleMessage(client *Client, msg models.WSMessage) {
	switch msg.Event {
	case "typing:start":
		h.handleTyping(client, msg, "typing:start")
	case "typing:stop":
		h.handleTyping(client, msg, "typing:stop")
	default:
		log.Printf("Unknown WS event: %s from userID=%d", msg.Event, client.UserID)
	}
}

func (h *Hub) handleTyping(client *Client, msg models.WSMessage, event string) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}
	chatID, _ := data["chatId"].(string)
	if chatID == "" {
		return
	}

	// Получаем участников и шлём всем кроме отправителя
	// Используем короткий контекст
	import_ctx_cancel := func() {}
	_ = import_ctx_cancel

	members, err := h.db.GetChatMembers(ctxBackground(), chatID)
	if err != nil {
		return
	}

	typingMsg := models.WSMessage{
		Event: event,
		Data:  map[string]interface{}{"chatId": chatID, "fromId": client.UserID},
	}
	for _, uid := range members {
		if uid != client.UserID {
			h.SendToUser(uid, typingMsg)
		}
	}
}

func (h *Hub) IsUserOnline(userID int) bool {
	_, exists := h.clients.Load(userID)
	return exists
}

func (h *Hub) GetClientCount() int {
	count := 0
	h.clients.Range(func(_, _ interface{}) bool { count++; return true })
	return count
}

func (h *Hub) Shutdown() {
	close(h.shutdown)
	h.wg.Wait()
}

func (h *Hub) shutdownAllClients() {
	h.clients.Range(func(_, v interface{}) bool {
		if c, ok := v.(*Client); ok {
			c.Close()
		}
		return true
	})
}
