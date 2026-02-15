// Gateway/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Gateway/config"
	"Gateway/handlers"
	"Gateway/middleware"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создание роутера
	r := mux.NewRouter()

	// Регистрация middleware
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	// Создание и регистрация обработчиков
	gatewayHandler := handlers.NewGatewayHandler(cfg)
	gatewayHandler.RegisterRoutes(r)

	// Статические файлы
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir(cfg.StaticDir)),
	))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowCredentials: cfg.CORS.AllowCredentials,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
	})

	handler := c.Handler(r)

	// HTTP сервер
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Gateway server starting on %s", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидание сигнала завершения
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
