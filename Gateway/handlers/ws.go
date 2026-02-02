package handlers

import (
	"Gateway/auth"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// login -> ws клиента
var clients = make(map[string]*websocket.Conn)

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// 1️⃣ JWT
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	login, err := auth.ValidateJWT(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 2️⃣ Upgrade клиента
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}
	log.Println("WS connected:", login)
	clients[login] = conn

	// 3️⃣ Подключение к Chat Service
	chatURL := url.URL{Scheme: "ws", Host: "localhost:8084", Path: "/ws", RawQuery: "login=" + login}
	chatConn, _, err := websocket.DefaultDialer.Dial(chatURL.String(), nil)
	if err != nil {
		log.Println("cannot connect to chat service:", err)
		conn.Close()
		delete(clients, login)
		return
	}

	// 4️⃣ Запуск go-рутин для пересылки сообщений от клиента в Chat Service
	go func() {
		defer chatConn.Close()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("client disconnected:", login)
				return
			}
			log.Printf("forward to chat service: %s", msg)
			chatConn.WriteMessage(mt, msg)
		}
	}()

	// 5️⃣ Пересылка сообщений из Chat Service клиенту
	go func() {
		defer conn.Close()
		for {
			_, msg, err := chatConn.ReadMessage()
			if err != nil {
				log.Println("chat service disconnected for", login)
				return
			}
			log.Printf("forward to client %s: %s", login, msg)
			conn.WriteMessage(websocket.TextMessage, msg)
		}
	}()
}
