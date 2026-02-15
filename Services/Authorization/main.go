// Authorization/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Authorization/config"
	"Authorization/db"
	"Authorization/handlers"
	"Authorization/middleware"

	"encoding/json"

	"github.com/gorilla/mux"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	data, _ := json.MarshalIndent(cfg.Database, "", "  ")
	log.Println(string(data))

	// Инициализация БД
	database, err := db.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Миграции
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Создание обработчиков
	authHandler := handlers.NewAuthHandler(cfg, database)

	// Создание роутера
	r := mux.NewRouter()

	// Middleware
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)
	//r.Use(middleware.CORS(cfg.CORS))

	// Регистрация маршрутов
	authHandler.RegisterRoutes(r)

	// Health check
	r.HandleFunc("/health", healthCheck).Methods("GET")

	// HTTP сервер
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Запуск сервера в горутине
	go func() {
		log.Printf("Auth service starting on %s", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"authorization"}`))
}
