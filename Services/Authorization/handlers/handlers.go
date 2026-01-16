package handlers

import (
	"Authorization/auth"
	"Authorization/db"
	"Authorization/models"
	"encoding/json"
	"net/http"
)

type request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// POST /register
func Register(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "error hashing password", 500)
		return
	}

	user := models.User{Login: req.Login, Password: hash}
	if err := db.CreateUser(user); err != nil {
		http.Error(w, "user exists or DB error", 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// POST /login
func Login(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	user, err := db.GetUserByLogin(req.Login)
	if err != nil || !auth.CheckPassword(user.Password, req.Password) {
		http.Error(w, "invalid login or password", 401)
		return
	}

	token, err := auth.GenerateJWT(user.Login)
	if err != nil {
		http.Error(w, "cannot generate token", 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
