package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ReminderRule struct {
	ID       uuid.UUID  `json:"id"`
	TaskID   uuid.UUID  `json:"task_id"`
	RemindAt time.Time  `json:"remind_at"`
	IsSent   bool       `json:"is_sent"`
	SentAt   *time.Time `json:"sent_at,omitempty"`
}

type Task struct {
	ID         uuid.UUID  `json:"id"`
	HouseID    uuid.UUID  `json:"house_id"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	Title      string     `json:"title"`
	Priority   string     `json:"priority"`
}

type TaskServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTaskServiceClient(baseURL string) *TaskServiceClient {
	return &TaskServiceClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *TaskServiceClient) GetPendingReminders(ctx context.Context) ([]ReminderRule, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/reminders/pending", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var reminders []ReminderRule
	if err := json.NewDecoder(resp.Body).Decode(&reminders); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return reminders, nil
}

func (c *TaskServiceClient) MarkReminderSent(ctx context.Context, reminderID uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/reminders/%s/sent", c.baseURL, reminderID), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func (c *TaskServiceClient) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*Task, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/tasks/%s", c.baseURL, taskID), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &task, nil
}
