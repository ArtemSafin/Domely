package config

import (
	"fmt"
	"os"
)

type Config struct {
	BotToken       string
	TaskServiceURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		BotToken:       getEnv("BOT_TOKEN", ""),
		TaskServiceURL: getEnv("TASK_SERVICE_URL", ""),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.TaskServiceURL == "" {
		return nil, fmt.Errorf("TASK_SERVICE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
