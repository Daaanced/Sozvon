// Gateway/config/config.go
package config

import (
	"os"
	"time"
)

type Config struct {
	Server    ServerConfig
	Services  ServicesConfig
	JWT       JWTConfig
	CORS      CORSConfig
	StaticDir string
}

type ServerConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type ServicesConfig struct {
	AuthServiceURL string
	UserServiceURL string
	ChatServiceURL string
}

type JWTConfig struct {
	SecretKey []byte
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	AllowedMethods   []string
	AllowedHeaders   []string
}

// Load загружает конфигурацию из переменных окружения с fallback на дефолтные значения
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Services: ServicesConfig{
			AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
			UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8083"),
			ChatServiceURL: getEnv("CHAT_SERVICE_URL", "http://localhost:8084"),
		},
		JWT: JWTConfig{
			SecretKey: []byte(getEnv("JWT_SECRET_KEY", "supersecretkey")),
		},
		CORS: CORSConfig{
			AllowedOrigins:   []string{getEnv("CORS_ALLOWED_ORIGINS", "*")},
			AllowCredentials: true,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
		},
		StaticDir: getEnv("STATIC_DIR", "../Services/User_Service/static/"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
