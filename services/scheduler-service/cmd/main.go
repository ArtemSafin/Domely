package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	robfigcron "github.com/robfig/cron/v3"
	"github.com/ArtemSafin/Domely/services/scheduler-service/internal/client"
	"github.com/ArtemSafin/Domely/services/scheduler-service/internal/config"
	"github.com/ArtemSafin/Domely/services/scheduler-service/internal/cron"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	taskClient := client.NewTaskServiceClient(cfg.TaskServiceURL)

	publisher, err := client.NewRedisPublisher(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis publisher: %v", err)
	}
	defer publisher.Close()

	scheduler := cron.NewScheduler(taskClient, publisher)

	c := robfigcron.New()
	if _, err := c.AddFunc(cfg.CronSchedule, scheduler.Run); err != nil {
		log.Fatalf("add cron func: %v", err)
	}

	c.Start()
	log.Printf("scheduler-service started, cron: %q", cfg.CronSchedule)

	scheduler.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx := c.Stop()
	<-ctx.Done()
	log.Println("stopped")
}
