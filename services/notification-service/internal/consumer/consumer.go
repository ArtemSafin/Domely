package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/ArtemSafin/Domely/services/notification-service/internal/telegram"
)

const (
	notificationQueue = "notifications"
	blockTimeout      = 5 * time.Second
)

type NotificationMessage struct {
	ReminderID uuid.UUID  `json:"reminder_id"`
	TaskID     uuid.UUID  `json:"task_id"`
	HouseID    uuid.UUID  `json:"house_id"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	Title      string     `json:"title"`
	Priority   string     `json:"priority"`
	// telegram_ids кому слать — заполняется если AssignedTo == nil (всем членам дома)
	TelegramIDs []int64 `json:"telegram_ids"`
}

type Consumer struct {
	rdb    *redis.Client
	sender *telegram.Sender
}

func New(redisURL string, sender *telegram.Sender) (*Consumer, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)

	// проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Consumer{rdb: rdb, sender: sender}, nil
}

// Run блокирующий цикл чтения из очереди
func (c *Consumer) Run(ctx context.Context) {
	log.Println("consumer: waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("consumer: context cancelled, stopping")
			return
		default:
		}

		// BRPOP блокируется до появления сообщения или таймаута
		result, err := c.rdb.BRPop(ctx, blockTimeout, notificationQueue).Result()
		if err != nil {
			if err == redis.Nil {
				// таймаут — очередь пустая, продолжаем ждать
				continue
			}
			if ctx.Err() != nil {
				// контекст отменён — выходим чисто
				return
			}
			log.Printf("consumer: brpop error: %v", err)
			time.Sleep(time.Second) // небольшая пауза перед retry
			continue
		}

		// result[0] = имя очереди, result[1] = payload
		if err := c.process(ctx, result[1]); err != nil {
			log.Printf("consumer: process message: %v", err)
		}
	}
}

func (c *Consumer) process(ctx context.Context, payload string) error {
	var msg NotificationMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	log.Printf("consumer: processing reminder %s, task %q, priority %s",
		msg.ReminderID, msg.Title, msg.Priority)

	if len(msg.TelegramIDs) == 0 {
		log.Printf("consumer: no recipients for reminder %s, skipping", msg.ReminderID)
		return nil
	}

	var sendErr error
	for _, telegramID := range msg.TelegramIDs {
		if err := c.sender.SendReminder(telegramID, msg.Title, msg.Priority); err != nil {
			// логируем но продолжаем — один недоставленный не блокирует остальных
			log.Printf("consumer: send to %d: %v", telegramID, err)
			sendErr = err
			continue
		}
		log.Printf("consumer: sent reminder to telegram_id=%d", telegramID)
	}

	return sendErr
}

func (c *Consumer) Close() error {
	return c.rdb.Close()
}
