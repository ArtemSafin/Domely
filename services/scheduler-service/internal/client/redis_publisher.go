package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const NotificationQueue = "notifications"

type NotificationMessage struct {
	ReminderID  uuid.UUID  `json:"reminder_id"`
	TaskID      uuid.UUID  `json:"task_id"`
	HouseID     uuid.UUID  `json:"house_id"`
	AssignedTo  *uuid.UUID `json:"assigned_to,omitempty"`
	Title       string     `json:"title"`
	Priority    string     `json:"priority"`
	TelegramIDs []int64    `json:"telegram_ids"`
}

type RedisPublisher struct {
	rdb *redis.Client
}

func NewRedisPublisher(redisURL string) (*RedisPublisher, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	return &RedisPublisher{rdb: rdb}, nil
}

func (p *RedisPublisher) Publish(ctx context.Context, msg NotificationMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	if err := p.rdb.LPush(ctx, NotificationQueue, data).Err(); err != nil {
		return fmt.Errorf("lpush: %w", err)
	}
	return nil
}

func (p *RedisPublisher) Close() error {
	return p.rdb.Close()
}
