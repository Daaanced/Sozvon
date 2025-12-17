package handlers

import (
	"encoding/json"
	"net/http"

	"dbserver/db"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.GetAllUsers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Считывает JSON-тело POST-запроса.
// Превращает JSON в структуру Go.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	var req request
	json.NewDecoder(r.Body).Decode(&req)

	err := db.CreateUser(req.Login, req.Password)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	w.Write([]byte(`{"status":"ok"}`))
}
