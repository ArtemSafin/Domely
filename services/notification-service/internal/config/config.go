package config

import (
	"fmt"
	"os"
)

type Config struct {
	RedisURL    string
	BotToken    string
}

func Load() (*Config, error) {
	cfg := &Config{
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		BotToken: getEnv("BOT_TOKEN", ""),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
