package main

import (
	"Authorization/db"
	"Authorization/handlers"

	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	db.Init()

	r := mux.NewRouter()
	r.HandleFunc("/register", handlers.Register).Methods("POST")
	r.HandleFunc("/login", handlers.Login).Methods("POST")

	log.Println("Auth service running on :8082")
	log.Fatal(http.ListenAndServe(":8082", r))
}
