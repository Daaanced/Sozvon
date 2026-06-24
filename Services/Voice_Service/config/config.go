// voice_service/config/config.go

package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address string

	// JWT — тот же секрет что у Gateway/Auth Service
	JWTSecret string

	// ICE серверы
	STUNServers []string
	TURNServer  string
	TURNUser    string
	TURNPass    string

	PublicIP string
	// UDP диапазон портов для WebRTC медиапотоков
	UDPPortMin uint16
	UDPPortMax uint16

	// Ограничения комнат
	MaxRooms       int
	MaxPeersInRoom int
}

func Load() *Config {
	return &Config{
		Address:   getEnv("VOICE_SERVICE_ADDRESS", ":8085"),
		JWTSecret: getEnv("JWT_SECRET_KEY", "supersecretkey"),

		STUNServers: strings.Split(
			getEnv("STUN_SERVERS", "stun:stun.sipnet.ru:3478"),
			",",
		),
		TURNServer: getEnv("TURN_SERVER", ""),
		TURNUser:   getEnv("TURN_USER", ""),
		TURNPass:   getEnv("TURN_PASS", ""),

		PublicIP:   os.Getenv("PUBLIC_IP"),
		UDPPortMin: uint16(getIntEnv("UDP_PORT_MIN", 10000)),
		UDPPortMax: uint16(getIntEnv("UDP_PORT_MAX", 10200)),

		MaxRooms:       getIntEnv("MAX_ROOMS", 100),
		MaxPeersInRoom: getIntEnv("MAX_PEERS_IN_ROOM", 16),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
