package config

import (
	"fmt"
	"os"
)

type Config struct {
	TaskServiceURL string
	RedisURL       string
	CronSchedule   string
}

func Load() (*Config, error) {
	cfg := &Config{
		TaskServiceURL: getEnv("TASK_SERVICE_URL", ""),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		CronSchedule:   getEnv("CRON_SCHEDULE", "* * * * *"),
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
