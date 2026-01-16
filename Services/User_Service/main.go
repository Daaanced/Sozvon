package main

import (
	"log"
	"net/http"

	"User_Service/db"
	"User_Service/handlers"

	"github.com/gorilla/mux"
)

func main() {
	db.Init()

	r := mux.NewRouter()

	r.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	r.HandleFunc("/users/{login}", handlers.GetUser).Methods("GET")
	r.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	r.HandleFunc("/users/{login}", handlers.UpdateUser).Methods("PUT")
	r.HandleFunc("/users/{login}", handlers.DeleteUser).Methods("DELETE")

	log.Println("User service running on :8083")
	log.Fatal(http.ListenAndServe(":8083", r))
}
