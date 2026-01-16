package handlers

import (
	"encoding/json"
	"net/http"

	"User_Service/db"
	"User_Service/models"

	"github.com/gorilla/mux"
)

// GET /users
func GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.GetAllUsers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GET /users/{login}
func GetUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]
	user, err := db.GetUserByLogin(login)
	if err != nil {
		http.Error(w, "user not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// POST /users
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	if err := db.CreateUser(u); err != nil {
		http.Error(w, "db error", 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// PUT /users/{login}
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	u.Login = login

	if err := db.UpdateUser(u); err != nil {
		http.Error(w, "db error", 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DELETE /users/{login}
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]

	if err := db.DeleteUser(login); err != nil {
		http.Error(w, "db error", 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
