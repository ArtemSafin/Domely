package cron

import (
	"context"
	"log"
	"time"

	"github.com/ArtemSafin/Domely/services/scheduler-service/internal/client"
)

type Scheduler struct {
	taskClient *client.TaskServiceClient
	publisher  *client.RedisPublisher
}

func NewScheduler(taskClient *client.TaskServiceClient, publisher *client.RedisPublisher) *Scheduler {
	return &Scheduler{taskClient: taskClient, publisher: publisher}
}

func (s *Scheduler) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	log.Println("scheduler: checking pending reminders...")

	reminders, err := s.taskClient.GetPendingReminders(ctx)
	if err != nil {
		log.Printf("scheduler: get pending reminders: %v", err)
		return
	}
	if len(reminders) == 0 {
		return
	}
	log.Printf("scheduler: found %d pending reminders", len(reminders))

	for _, reminder := range reminders {
		if err := s.processReminder(ctx, reminder); err != nil {
			log.Printf("scheduler: process reminder %s: %v", reminder.ID, err)
			continue
		}
	}
}

func (s *Scheduler) processReminder(ctx context.Context, reminder client.ReminderRule) error {
	task, err := s.taskClient.GetTaskByID(ctx, reminder.TaskID)
	if err != nil {
		return err
	}

	msg := client.NotificationMessage{
		ReminderID: reminder.ID,
		TaskID:     task.ID,
		HouseID:    task.HouseID,
		AssignedTo: task.AssignedTo,
		Title:      task.Title,
		Priority:   task.Priority,
	}

	if err := s.publisher.Publish(ctx, msg); err != nil {
		return err
	}

	if err := s.taskClient.MarkReminderSent(ctx, reminder.ID); err != nil {
		log.Printf("scheduler: mark reminder sent %s: %v", reminder.ID, err)
	}

	log.Printf("scheduler: queued reminder %s for task %q", reminder.ID, task.Title)
	return nil
}
