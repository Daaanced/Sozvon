// Chat_Service\handlers\ws.go
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

var (
	clients = make(map[string]*websocket.Conn)
	mu      sync.Mutex
)

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

	// register client
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

		if msg.Event != "message:send" {
			continue
		}

		// 1️⃣ load chat
		storage.Mu.Lock()
		chat, ok := storage.Chats[msg.Data.ChatID]
		if !ok {
			storage.Mu.Unlock()
			log.Println("chat not found:", msg.Data.ChatID)
			continue
		}

		// 2️⃣ activate chat ONCE (first message)
		justActivated := false
		if !chat.Active {
			chat.Active = true
			storage.Chats[msg.Data.ChatID] = chat
			justActivated = true
		}
		storage.Mu.Unlock()

		// 3️⃣ notify second participant ONLY ONCE
		if justActivated {
			for _, member := range chat.Members {
				mu.Lock()
				conn, ok := clients[member]
				mu.Unlock()

				if !ok {
					continue
				}

				conn.WriteJSON(map[string]interface{}{
					"event": "chat:created",
					"data": map[string]interface{}{
						"chatId":  chat.ID,
						"members": chat.Members,
					},
				})

				log.Printf("chat:created sent to %s (chat=%s)", member, chat.ID)
			}
		}

		// 4️⃣ deliver message to all online members
		for _, member := range chat.Members {
			mu.Lock()
			receiver, ok := clients[member]
			mu.Unlock()
			if !ok {
				continue
			}

			receiver.WriteJSON(map[string]interface{}{
				"event": "message:new",
				"data": map[string]string{
					"chatId": chat.ID,
					"from":   login,
					"text":   msg.Data.Text,
				},
			})
		}
	}
}
