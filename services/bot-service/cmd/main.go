package main

import (
	"log"
	"time"

	tele "gopkg.in/telebot.v3"
	"github.com/ArtemSafin/Domely/services/bot-service/internal/client"
	"github.com/ArtemSafin/Domely/services/bot-service/internal/config"
	"github.com/ArtemSafin/Domely/services/bot-service/internal/handler"
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

	taskClient := client.NewTaskServiceClient(cfg.TaskServiceURL)
	h := handler.New(bot, taskClient)
	h.Register()

	log.Println("bot-service started")
	bot.Start()
}
