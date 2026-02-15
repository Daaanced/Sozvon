// Gateway/handlers/gateway.go
package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"Gateway/config"

	"github.com/gorilla/mux"
)

// ProxyClient интерфейс для проксирования запросов
type ProxyClient interface {
	ProxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) error
}

// GatewayHandler обрабатывает маршрутизацию запросов к микросервисам
type GatewayHandler struct {
	config     *config.Config
	httpClient *http.Client
	wsHandler  *WebSocketHandler
}

// NewGatewayHandler создает новый экземпляр GatewayHandler
func NewGatewayHandler(cfg *config.Config) *GatewayHandler {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &GatewayHandler{
		config:     cfg,
		httpClient: httpClient,
		wsHandler:  NewWebSocketHandler(cfg),
	}
}

// RegisterRoutes регистрирует все маршруты Gateway
func (h *GatewayHandler) RegisterRoutes(r *mux.Router) {
	// Auth Service
	r.PathPrefix("/auth/").HandlerFunc(h.proxyToAuth)

	// User Service
	r.PathPrefix("/users/").HandlerFunc(h.proxyToUsers)

	// Chat Service
	r.PathPrefix("/chats").HandlerFunc(h.proxyToChats)

	// WebSocket
	r.HandleFunc("/ws", h.wsHandler.HandleWebSocket)

	// Health check
	r.HandleFunc("/health", h.healthCheck).Methods("GET")
}

// proxyToAuth проксирует запросы к Auth Service
func (h *GatewayHandler) proxyToAuth(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.config.Services.AuthServiceURL)
}

// proxyToUsers проксирует запросы к User Service
func (h *GatewayHandler) proxyToUsers(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.config.Services.UserServiceURL)
}

// proxyToChats проксирует запросы к Chat Service
func (h *GatewayHandler) proxyToChats(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.config.Services.ChatServiceURL)
}

// proxyRequest проксирует HTTP запрос к целевому сервису
func (h *GatewayHandler) proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Формируем URL целевого сервиса
	targetReq, err := http.NewRequestWithContext(
		ctx,
		r.Method,
		targetURL+r.RequestURI,
		r.Body,
	)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Копируем заголовки от клиента
	h.copyHeaders(targetReq.Header, r.Header)

	// Отправляем запрос к целевому сервису
	resp, err := h.httpClient.Do(targetReq)
	if err != nil {
		log.Printf("Error proxying request to %s: %v", targetURL, err)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	h.copyHeaders(w.Header(), resp.Header)

	// Устанавливаем статус код
	w.WriteHeader(resp.StatusCode)

	// Копируем тело ответа
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}

// copyHeaders копирует заголовки из src в dst
func (h *GatewayHandler) copyHeaders(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

// healthCheck endpoint для проверки состояния Gateway
func (h *GatewayHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"gateway"}`))
}
