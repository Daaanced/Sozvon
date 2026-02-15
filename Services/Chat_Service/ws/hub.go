// Chat_Service/ws/hub.go
package ws

import (
	"context"
	"log"
	"sync"
	"time"

	"Chat_Service/config"
	"Chat_Service/db"
	"Chat_Service/models"
)

// Hub управляет WebSocket клиентами и рассылкой сообщений
type Hub struct {
	config     *config.Config
	db         *db.Database
	clients    sync.Map // login -> *Client
	Register   chan *Client
	Unregister chan *Client
	broadcast  chan *BroadcastMessage
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

// BroadcastMessage сообщение для рассылки
type BroadcastMessage struct {
	Recipients []string
	Message    models.WSMessage
}

// NewHub создает новый WebSocket hub
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

// Run запускает главный цикл hub
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

// registerClient регистрирует нового клиента
func (h *Hub) registerClient(client *Client) {
	// Проверка на существующее подключение
	if existing, exists := h.clients.LoadAndDelete(client.Login); exists {
		if c, ok := existing.(*Client); ok {
			log.Printf("Closing existing connection for %s", client.Login)
			c.Close()
		}
	}

	h.clients.Store(client.Login, client)
	log.Printf("Client registered: %s (total: %d)", client.Login, h.GetClientCount())
}

// unregisterClient отключает клиента
func (h *Hub) unregisterClient(client *Client) {
	if _, exists := h.clients.LoadAndDelete(client.Login); exists {
		log.Printf("Client unregistered: %s (total: %d)", client.Login, h.GetClientCount())
	}
}

// broadcastMessage рассылает сообщение получателям
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	for _, recipient := range msg.Recipients {
		if client, ok := h.clients.Load(recipient); ok {
			if c, ok := client.(*Client); ok {
				select {
				case c.send <- msg.Message:
					// Сообщение отправлено
				default:
					// Канал переполнен, пропускаем
					log.Printf("Warning: send buffer full for %s", recipient)
				}
			}
		}
	}
}

// SendToUser отправляет сообщение конкретному пользователю
func (h *Hub) SendToUser(login string, message models.WSMessage) {
	if client, ok := h.clients.Load(login); ok {
		if c, ok := client.(*Client); ok {
			select {
			case c.send <- message:
			default:
				log.Printf("Warning: failed to send to %s (buffer full)", login)
			}
		}
	}
}

// SendToUsers отправляет сообщение нескольким пользователям
func (h *Hub) SendToUsers(logins []string, message models.WSMessage) {
	h.broadcast <- &BroadcastMessage{
		Recipients: logins,
		Message:    message,
	}
}

// HandleMessage обрабатывает входящее сообщение от клиента
func (h *Hub) HandleMessage(client *Client, msg models.WSMessage) {
	switch msg.Event {
	case "message:send":
		h.handleSendMessage(client, msg)
	case "typing:start":
		h.handleTypingStart(client, msg)
	case "typing:stop":
		h.handleTypingStop(client, msg)
	default:
		log.Printf("Unknown event: %s from %s", msg.Event, client.Login)
	}
}

// handleSendMessage обрабатывает отправку сообщения
func (h *Hub) handleSendMessage(client *Client, msg models.WSMessage) {
	// Парсинг данных
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid message data from %s", client.Login)
		return
	}

	chatID, _ := data["chatId"].(string)
	text, _ := data["text"].(string)

	// Валидация
	sendReq := models.SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	if err := sendReq.Validate(); err != nil {
		client.SendError("validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Получение участников чата
	members, err := h.db.GetChatMembers(ctx, chatID)
	if err != nil {
		log.Printf("Error getting chat members: %v", err)
		client.SendError("database_error", "Failed to get chat members")
		return
	}

	if len(members) == 0 {
		client.SendError("chat_not_found", "Chat not found")
		return
	}

	// Проверка что отправитель является участником
	isMember := false
	for _, member := range members {
		if member == client.Login {
			isMember = true
			break
		}
	}
	if !isMember {
		client.SendError("forbidden", "You are not a member of this chat")
		return
	}

	// Сохранение сообщения
	messageID, err := h.db.SaveMessage(ctx, chatID, client.Login, text)
	if err != nil {
		log.Printf("Error saving message: %v", err)
		client.SendError("database_error", "Failed to save message")
		return
	}

	// Активация чата (если это первое сообщение)
	activated, err := h.db.ActivateChat(ctx, chatID)
	if err != nil {
		log.Printf("Error activating chat: %v", err)
	}

	// Уведомление об активации
	if activated {
		activationMsg := models.WSMessage{
			Event: "chat:created",
			Data: map[string]string{
				"chatId": chatID,
			},
		}
		h.SendToUsers(members, activationMsg)
	}

	// Рассылка сообщения всем участникам
	newMessageEvent := models.WSMessage{
		Event: "message:new",
		Data: map[string]interface{}{
			"id":        messageID,
			"chatId":    chatID,
			"from":      client.Login,
			"text":      text,
			"createdAt": time.Now().Format(time.RFC3339),
		},
	}
	h.SendToUsers(members, newMessageEvent)
}

// handleTypingStart обрабатывает начало набора текста
func (h *Hub) handleTypingStart(client *Client, msg models.WSMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	chatID, _ := data["chatId"].(string)
	if chatID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	members, err := h.db.GetChatMembers(ctx, chatID)
	if err != nil {
		return
	}

	// Отправка уведомления всем кроме отправителя
	typingMsg := models.WSMessage{
		Event: "typing:start",
		Data: map[string]string{
			"chatId": chatID,
			"from":   client.Login,
		},
	}

	for _, member := range members {
		if member != client.Login {
			h.SendToUser(member, typingMsg)
		}
	}
}

// handleTypingStop обрабатывает окончание набора текста
func (h *Hub) handleTypingStop(client *Client, msg models.WSMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	chatID, _ := data["chatId"].(string)
	if chatID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	members, err := h.db.GetChatMembers(ctx, chatID)
	if err != nil {
		return
	}

	typingMsg := models.WSMessage{
		Event: "typing:stop",
		Data: map[string]string{
			"chatId": chatID,
			"from":   client.Login,
		},
	}

	for _, member := range members {
		if member != client.Login {
			h.SendToUser(member, typingMsg)
		}
	}
}

// GetClientCount возвращает количество подключенных клиентов
func (h *Hub) GetClientCount() int {
	count := 0
	h.clients.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// IsUserOnline проверяет онлайн ли пользователь
func (h *Hub) IsUserOnline(login string) bool {
	_, exists := h.clients.Load(login)
	return exists
}

// Shutdown корректно завершает работу hub
func (h *Hub) Shutdown() {
	log.Println("Shutting down WebSocket hub...")
	close(h.shutdown)
	h.wg.Wait()
	log.Println("WebSocket hub shutdown complete")
}

// shutdownAllClients закрывает все клиентские соединения
func (h *Hub) shutdownAllClients() {
	h.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			client.Close()
		}
		return true
	})
}
