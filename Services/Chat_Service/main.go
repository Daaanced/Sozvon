// Chat_Service\main.go
package main

import (
	"log"
	"net/http"

	"Chat_Service/db"
	"Chat_Service/handlers"
)

func main() {

	db.InitDB()

	http.HandleFunc("/ws", handlers.HandleWS)
	http.HandleFunc("/chats/create", handlers.CreateChat)
	http.HandleFunc("/chats", handlers.GetChats)
	http.HandleFunc("/chats/", handlers.GetMessages)

	log.Println("Chat Service running on :8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}
