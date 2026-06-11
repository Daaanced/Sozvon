// Chat_Service\handlers\ws.go
package handlers

import (
	"log"
	"net/http"

	"Chat_Service/ws"
)

func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID
	if err != nil || userID == 0 {
		http.Error(w, "invalid user_id in token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, userID)
	h.hub.Register <- client
	client.Start()
}
