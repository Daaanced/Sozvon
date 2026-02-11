// Chat_Service\handlers\ws.go
package handlers

import (
	"Chat_Service/db"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
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
	// ⚠️ ВАЖНО: login, а не token
	login := r.URL.Query().Get("login")
	if login == "" {
		http.Error(w, "login required", 400)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

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

		// 1️⃣ получаем участников чата
		rows, err := db.DB.Query(`
			SELECT login FROM chat_members WHERE chat_id = $1
		`, msg.Data.ChatID)
		if err != nil {
			continue
		}

		var members []string
		for rows.Next() {
			var m string
			rows.Scan(&m)
			members = append(members, m)
		}
		rows.Close()

		if len(members) == 0 {
			continue
		}

		// 2️⃣ сохраняем сообщение
		_, err = db.DB.Exec(`
			INSERT INTO messages (id, chat_id, sender_login, text)
			VALUES ($1, $2, $3, $4)
		`, uuid.NewString(), msg.Data.ChatID, login, msg.Data.Text)

		if err != nil {
			log.Println("insert message error:", err)
			continue
		}

		// 3️⃣ АКТИВАЦИЯ ЧАТА (ОДИН РАЗ)
		var justActivated bool
		err = db.DB.QueryRow(`
			UPDATE chats
			SET active = true
			WHERE id = $1 AND active = false
			RETURNING true
		`, msg.Data.ChatID).Scan(&justActivated)

		if err == nil && justActivated {
			for _, member := range members {
				mu.Lock()
				c, ok := clients[member]
				mu.Unlock()

				if ok {
					c.WriteJSON(map[string]interface{}{
						"event": "chat:created",
						"data": map[string]string{
							"chatId": msg.Data.ChatID,
						},
					})
				}
			}
		}

		// 4️⃣ доставляем сообщение
		for _, member := range members {
			mu.Lock()
			c, ok := clients[member]
			mu.Unlock()

			if ok {
				c.WriteJSON(map[string]interface{}{
					"event": "message:new",
					"data": map[string]string{
						"chatId": msg.Data.ChatID,
						"from":   login,
						"text":   msg.Data.Text,
					},
				})
			}
		}
	}
}
