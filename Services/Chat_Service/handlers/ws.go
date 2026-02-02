//Chat_Service\handlers\ws.go

package handlers

import (
	"Chat_Service/storage"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[string]*websocket.Conn)
var mu sync.Mutex

type WSMessage struct {
	Event string `json:"event"`
	Data  struct {
		ChatID string `json:"chatId"`
		Text   string `json:"text"`
	} `json:"data"`
}

func HandleWS(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if login == "" {
		http.Error(w, "login required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	// регистрируем клиента
	mu.Lock()
	clients[login] = conn
	mu.Unlock()
	log.Println("WS connected:", login)

	defer func() {
		mu.Lock()
		delete(clients, login)
		mu.Unlock()
		conn.Close()
		log.Println("WS disconnected:", login)
	}()

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		log.Printf("WS message from %s: %+v\n", login, msg)

		// 1️⃣ найти чат
		storage.Mu.Lock()
		chat, ok := storage.Chats[msg.Data.ChatID]
		storage.Mu.Unlock()

		if !ok {
			log.Println("chat not found:", msg.Data.ChatID)
			continue
		}

		// 2️⃣ отправить всем участникам (кроме отправителя)
		for _, member := range chat.Members {
			mu.Lock()
			receiver, ok := clients[member]
			mu.Unlock()
			if !ok {
				log.Printf("receiver offline: %s\n", member)
				continue
			}

			err := receiver.WriteJSON(map[string]interface{}{
				"event": "message:new",
				"data": map[string]string{
					"chatId": msg.Data.ChatID,
					"from":   login,
					"text":   msg.Data.Text,
				},
			})

			if err != nil {
				log.Println("send error to", member, err)
			} else {
				log.Printf("sending message to %s: chatId=%s, text=%s", member, msg.Data.ChatID, msg.Data.Text)
			}
		}

	}
}

func handleSendMessage(sender string, msg WSMessage) {
	chatID := msg.Data.ChatID
	text := msg.Data.Text

	// 1️⃣ найти чат
	storage.Mu.Lock()
	chat, ok := storage.Chats[chatID]
	storage.Mu.Unlock()

	if !ok {
		log.Println("chat not found:", chatID)
		return
	}

	// 2️⃣ найти получателя
	var receiverLogin string
	for _, m := range chat.Members {
		if m != sender {
			receiverLogin = m
		}
	}

	// 3️⃣ найти websocket получателя
	mu.Lock()
	receiverConn, ok := clients[receiverLogin]
	mu.Unlock()

	if !ok {
		log.Println("receiver offline:", receiverLogin)
		return
	}

	// 4️⃣ отправить сообщение
	receiverConn.WriteJSON(map[string]string{
		"from":   sender,
		"text":   text,
		"chatId": chatID,
	})
}
