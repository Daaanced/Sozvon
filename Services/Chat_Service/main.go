// Chat_Service\main.go
package main

import (
	"log"
	"net/http"

	"Chat_Service/handlers"
)

func main() {
	http.HandleFunc("/ws", handlers.HandleWS)
	http.HandleFunc("/chats/create", handlers.CreateChat)

	log.Println("Chat Service running on :8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}
