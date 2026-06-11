package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Name       string    `json:"name"`
}

type House struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	OwnerID uuid.UUID `json:"owner_id"`
}

type Task struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	TaskType     string     `json:"task_type"`
	Priority     string     `json:"priority"`
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	IntervalDays *int       `json:"interval_days,omitempty"`
	IsActive     bool       `json:"is_active"`
}

type CreateTaskRequest struct {
	HouseID          uuid.UUID  `json:"house_id"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	AssignedTo       *uuid.UUID `json:"assigned_to,omitempty"`
	Title            string     `json:"title"`
	TaskType         string     `json:"task_type"`
	Priority         string     `json:"priority"`
	ReminderStrategy string     `json:"reminder_strategy"`
	DueAt            *time.Time `json:"due_at,omitempty"`
	IntervalDays     *int       `json:"interval_days,omitempty"`
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

// --- Users ---

func (c *TaskServiceClient) RegisterUser(ctx context.Context, telegramID int64, name string) (*User, error) {
	body := map[string]interface{}{
		"telegram_id": telegramID,
		"name":        name,
	}
	var user User
	if err := c.post(ctx, "/users", body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *TaskServiceClient) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	var user User
	if err := c.get(ctx, fmt.Sprintf("/users/telegram/%d", telegramID), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// --- Houses ---

func (c *TaskServiceClient) CreateHouse(ctx context.Context, name string, ownerID uuid.UUID) (*House, error) {
	body := map[string]interface{}{
		"name":     name,
		"owner_id": ownerID,
	}
	var house House
	if err := c.post(ctx, "/houses", body, &house); err != nil {
		return nil, err
	}
	return &house, nil
}

func (c *TaskServiceClient) GetHousesByUser(ctx context.Context, userID uuid.UUID) ([]House, error) {
	var houses []House
	if err := c.get(ctx, fmt.Sprintf("/houses/user/%s", userID), &houses); err != nil {
		return nil, err
	}
	return houses, nil
}

func (c *TaskServiceClient) GetHouseMembers(ctx context.Context, houseID uuid.UUID) ([]User, error) {
	var members []User
	if err := c.get(ctx, fmt.Sprintf("/houses/%s/members", houseID), &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (c *TaskServiceClient) InviteMember(ctx context.Context, houseID, userID uuid.UUID) error {
	body := map[string]interface{}{"user_id": userID}
	return c.post(ctx, fmt.Sprintf("/houses/%s/members", houseID), body, nil)
}

// --- Tasks ---

func (c *TaskServiceClient) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var task Task
	if err := c.post(ctx, "/tasks", req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *TaskServiceClient) GetTasksByHouse(ctx context.Context, houseID uuid.UUID) ([]Task, error) {
	var tasks []Task
	if err := c.get(ctx, fmt.Sprintf("/tasks/house/%s", houseID), &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *TaskServiceClient) CompleteTask(ctx context.Context, taskID, userID uuid.UUID) error {
	body := map[string]interface{}{"completed_by": userID}
	return c.post(ctx, fmt.Sprintf("/tasks/%s/complete", taskID), body, nil)
}

func (c *TaskServiceClient) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/tasks/%s", c.baseURL, taskID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// --- HTTP helpers ---

func (c *TaskServiceClient) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s%s", c.baseURL, path), nil)
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

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *TaskServiceClient) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s%s", c.baseURL, path), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
