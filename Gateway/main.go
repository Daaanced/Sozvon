package main

import (
	"net/http"

	"Gateway/handlers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	r := mux.NewRouter()
	handlers.RegisterRoutes(r)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})

	handler := c.Handler(r)
	http.ListenAndServe(":8080", handler)
}
