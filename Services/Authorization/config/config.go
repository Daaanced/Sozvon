// Authorization/config/config.go
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	UserService UserServiceConfig
	CORS        CORSConfig
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
	SecretKey      []byte
	TokenDuration  time.Duration
	RefreshEnabled bool
}

type UserServiceConfig struct {
	URL     string
	Timeout time.Duration
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
			Address:      getEnv("SERVER_ADDRESS", ":8082"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getIntEnv("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_AUTH_NAME", "authdb"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			SecretKey:      []byte(getEnv("JWT_SECRET_KEY", "supersecretkey")),
			TokenDuration:  getDurationEnv("JWT_TOKEN_DURATION", 24*time.Hour),
			RefreshEnabled: getBoolEnv("JWT_REFRESH_ENABLED", false),
		},
		UserService: UserServiceConfig{
			URL:     getEnv("USER_SERVICE_URL", "http://localhost:8083"),
			Timeout: getDurationEnv("USER_SERVICE_TIMEOUT", 10*time.Second),
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

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
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
