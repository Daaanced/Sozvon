// User_Service/config/config.go
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Static   StaticConfig
	Storage  StorageConfig
	CORS     CORSConfig
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

type StaticConfig struct {
	Directory     string
	AvatarsPath   string
	DefaultAvatar string
	MaxUploadSize int64
}

type StorageConfig struct {
	BackendURL string
	CDNEnabled bool
	CDNURL     string
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
			Address:      getEnv("SERVER_ADDRESS", ":8083"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getIntEnv("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "userdb"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Static: StaticConfig{
			Directory:     getEnv("STATIC_DIR", "./static"),
			AvatarsPath:   getEnv("AVATARS_PATH", "/static/avatars/"),
			DefaultAvatar: getEnv("DEFAULT_AVATAR", "default.png"),
			MaxUploadSize: getInt64Env("MAX_UPLOAD_SIZE", 5*1024*1024), // 5MB
		},
		Storage: StorageConfig{
			BackendURL: getEnv("BACKEND_URL", "http://176.51.121.88:8080"),
			CDNEnabled: getBoolEnv("CDN_ENABLED", false),
			CDNURL:     getEnv("CDN_URL", ""),
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
