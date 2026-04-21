package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"Voice_Service/auth"
	"Voice_Service/config"
	"Voice_Service/sfu"

	"github.com/gorilla/mux"
)

type RoomHandler struct {
	engine     *sfu.Engine
	jwtService *auth.JWTService
}

func NewRoomHandler(cfg *config.Config, engine *sfu.Engine) *RoomHandler {
	return &RoomHandler{
		engine:     engine,
		jwtService: auth.NewJWTService(cfg.JWTSecret),
	}
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms := h.engine.ListRooms()
	writeJSON(w, http.StatusOK, map[string]any{
		"rooms": rooms,
		"total": len(rooms),
	})
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req sfu.RoomCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	room, err := h.engine.CreateRoom(req.Name, userID)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, sfu.RoomCreateResponse{
		RoomID: room.ID,
		Name:   room.Name,
	})
}

func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := mux.Vars(r)["roomID"]
	room, ok := h.engine.GetRoom(roomID)
	if !ok {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, room.Info())
}

func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomID := mux.Vars(r)["roomID"]
	if err := h.engine.DeleteRoom(roomID); err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RoomHandler) userIDFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no authorization header")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtService.ValidateToken(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
