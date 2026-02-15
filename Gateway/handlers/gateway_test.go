// Gateway/handlers/gateway_test.go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Gateway/config"
)

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Address:      ":8080",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Services: config.ServicesConfig{
			AuthServiceURL: "http://localhost:8082",
			UserServiceURL: "http://localhost:8083",
			ChatServiceURL: "http://localhost:8084",
		},
		JWT: config.JWTConfig{
			SecretKey: []byte("test-secret-key"),
		},
	}

	handler := NewGatewayHandler(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.healthCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestCopyHeaders(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey: []byte("test-secret-key"),
		},
	}

	handler := NewGatewayHandler(cfg)

	src := http.Header{}
	src.Add("Content-Type", "application/json")
	src.Add("X-Custom-Header", "value1")
	src.Add("X-Custom-Header", "value2")

	dst := http.Header{}
	handler.copyHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type not copied correctly")
	}

	customHeaders := dst.Values("X-Custom-Header")
	if len(customHeaders) != 2 {
		t.Errorf("Expected 2 values for X-Custom-Header, got %d", len(customHeaders))
	}
}

// Benchmark для проверки производительности
func BenchmarkHealthCheck(b *testing.B) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey: []byte("test-secret-key"),
		},
	}

	handler := NewGatewayHandler(cfg)
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.healthCheck(w, req)
	}
}
