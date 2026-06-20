// Voice_Service\main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Voice_Service/config"
	"Voice_Service/handlers"
	"Voice_Service/middleware"
	"Voice_Service/sfu"

	"github.com/gorilla/mux"
)

func main() {
	cfg := config.Load()

	sfuEngine := sfu.NewEngine(cfg)

	r := mux.NewRouter()
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	wsHandler := handlers.NewVoiceWSHandler(cfg, sfuEngine)
	roomHandler := handlers.NewRoomHandler(cfg, sfuEngine)

	// REST — управление комнатами
	r.HandleFunc("/api/voice/rooms", roomHandler.ListRooms).Methods("GET")
	r.HandleFunc("/api/voice/rooms", roomHandler.CreateRoom).Methods("POST")
	r.HandleFunc("/api/voice/rooms/{roomID}", roomHandler.GetRoom).Methods("GET")
	r.HandleFunc("/api/voice/rooms/{roomID}", roomHandler.DeleteRoom).Methods("DELETE")

	// WS — сигнализация (gateway проксирует сюда /ws/voice)
	r.HandleFunc("/ws", wsHandler.Handle)

	// Health
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"voice"}`))
	}).Methods("GET")

	// c := cors.New(cors.Options{
	// 	AllowedOrigins:   []string{"*"},
	// 	AllowCredentials: true,
	// 	AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
	// 	AllowedHeaders:   []string{"*"},
	// })

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Voice Service starting on %s", cfg.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Voice service error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Voice Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sfuEngine.Close()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Voice service forced shutdown: %v", err)
	}

	log.Println("Voice Service exited")
}
