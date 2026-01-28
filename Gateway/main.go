// Gateway/main.go
package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	AuthServiceURL = "http://localhost:8082"
	UserServiceURL = "http://localhost:8083"
)

func main() {
	log.Println("🚀 GATEWAY STARTED 8080")

	r := mux.NewRouter()

	// Test endpoint
	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		handleCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("🔥 TEST ENDPOINT HIT")
		w.WriteHeader(200)
		w.Write([]byte("I AM THE REAL GATEWAY"))
	})

	// Auth proxy
	r.PathPrefix("/auth/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		ProxyRequest(w, r, AuthServiceURL)
	})

	// User proxy
	r.PathPrefix("/users/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		ProxyRequest(w, r, UserServiceURL)
	})

	// Static files (если нужны)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("../Services/User_Service/static/"))))

	log.Println("🌐 Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// ProxyRequest отправляет запрос к сервису и возвращает ответ
func ProxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	req, err := http.NewRequest(r.Method, targetURL+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "cannot create request", 500)
		return
	}

	// Копируем заголовки клиента
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

	// Копируем заголовки и статус
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleCORS добавляет заголовки CORS для фронтенда
func handleCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "http://localhost:3000" || origin == "http://90.189.252.24:3000" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	}
	log.Println("🔥 CORS HANDLER CALLED", r.Method, r.URL.Path, "Origin:", origin)
}
