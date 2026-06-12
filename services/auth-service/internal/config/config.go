package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   string // например "24h"
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:    getEnv("HTTP_PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTExpiry:   getEnv("JWT_EXPIRY", "24h"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}