// Chat_Service/config/config.go
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	WebSocket WebSocketConfig
	CORS      CORSConfig
}

type ServerConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	SecretKey []byte
}

type WebSocketConfig struct {
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	AllowedMethods   []string
	AllowedHeaders   []string
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8084"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getIntEnv("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "chatdb"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			SecretKey: []byte(getEnv("JWT_SECRET_KEY", "supersecretkey")),
		},
		WebSocket: WebSocketConfig{
			WriteWait:       getDurationEnv("WS_WRITE_WAIT", 10*time.Second),
			PongWait:        getDurationEnv("WS_PONG_WAIT", 60*time.Second),
			PingPeriod:      getDurationEnv("WS_PING_PERIOD", 54*time.Second),
			MaxMessageSize:  getInt64Env("WS_MAX_MESSAGE_SIZE", 512*1024), // 512KB
			ReadBufferSize:  getIntEnv("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize: getIntEnv("WS_WRITE_BUFFER_SIZE", 1024),
		},
		CORS: CORSConfig{
			AllowedOrigins:   []string{getEnv("CORS_ALLOWED_ORIGINS", "*")},
			AllowCredentials: true,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
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
