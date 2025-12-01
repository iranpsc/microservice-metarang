package config

import (
	"os"
)

type Config struct {
	HTTPPort        string
	AuthServiceAddr string
}

func Load() *Config {
	return &Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		AuthServiceAddr: getEnv("AUTH_SERVICE_ADDR", "auth-service:50051"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

