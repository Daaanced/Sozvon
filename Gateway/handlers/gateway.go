// Gateway\handlers\gateway.go
package handlers

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// Адреса микросервисов
const (
	AuthServiceURL = "http://localhost:8082"
	UserServiceURL = "http://localhost:8083"
)

// Прокси-запрос к другому сервису
func ProxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	req, err := http.NewRequest(r.Method, targetURL+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "cannot create request", 500)
		return
	}

	// копируем headers клиента
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "service unavailable", 503)
		return
	}
	defer resp.Body.Close()

	// 🔥 КОПИРУЕМ ВСЕ HEADERS ОТ СЕРВИСА
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Маршруты Gateway
func RegisterRoutes(r *mux.Router) {
	// Auth Service
	r.PathPrefix("/auth/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProxyRequest(w, r, AuthServiceURL)
	})

	// User Service (требует токен JWT)
	r.PathPrefix("/users/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProxyRequest(w, r, UserServiceURL)
	})

}
