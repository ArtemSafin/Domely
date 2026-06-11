package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v3"
	"github.com/ArtemSafin/Domely/services/notification-service/internal/config"
	"github.com/ArtemSafin/Domely/services/notification-service/internal/consumer"
	"github.com/ArtemSafin/Domely/services/notification-service/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("telegram bot: %v", err)
	}

	sender := telegram.NewSender(bot)

	c, err := consumer.New(cfg.RedisURL, sender)
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down...")
		cancel()
	}()

	log.Println("notification-service started")
	c.Run(ctx)
	log.Println("stopped")
}
