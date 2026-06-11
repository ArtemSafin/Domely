package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskType string
type TaskPriority string
type ReminderStrategy string

const (
	TaskTypeRecurring TaskType = "recurring"
	TaskTypeOneTime   TaskType = "one_time"

	PriorityLow    TaskPriority = "low"
	PriorityNormal TaskPriority = "normal"
	PriorityHigh   TaskPriority = "high"

	ReminderSimple  ReminderStrategy = "simple"
	ReminderAdvance ReminderStrategy = "advance"
	ReminderMeeting ReminderStrategy = "meeting"
)

type User struct {
	ID         uuid.UUID `db:"id" json:"id"`
	TelegramID int64     `db:"telegram_id" json:"telegram_id"`
	Name       string    `db:"name" json:"name"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type House struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	OwnerID   uuid.UUID `db:"owner_id" json:"owner_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type HouseMember struct {
	HouseID uuid.UUID `db:"house_id" json:"house_id"`
	UserID  uuid.UUID `db:"user_id" json:"user_id"`
	Role    string    `db:"role" json:"role"`
}

type Task struct {
	ID               uuid.UUID        `db:"id" json:"id"`
	HouseID          uuid.UUID        `db:"house_id" json:"house_id"`
	CreatedBy        uuid.UUID        `db:"created_by" json:"created_by"`
	AssignedTo       *uuid.UUID       `db:"assigned_to" json:"assigned_to,omitempty"`
	Title            string           `db:"title" json:"title"`
	TaskType         TaskType         `db:"task_type" json:"task_type"`
	Priority         TaskPriority     `db:"priority" json:"priority"`
	ReminderStrategy ReminderStrategy `db:"reminder_strategy" json:"reminder_strategy"`
	DueAt            *time.Time       `db:"due_at" json:"due_at,omitempty"`
	NextRunAt        *time.Time       `db:"next_run_at" json:"next_run_at,omitempty"`
	IntervalDays     *int             `db:"interval_days" json:"interval_days,omitempty"`
	IsActive         bool             `db:"is_active" json:"is_active"`
	CreatedAt        time.Time        `db:"created_at" json:"created_at"`
}

type ReminderRule struct {
	ID       uuid.UUID  `db:"id" json:"id"`
	TaskID   uuid.UUID  `db:"task_id" json:"task_id"`
	RemindAt time.Time  `db:"remind_at" json:"remind_at"`
	IsSent   bool       `db:"is_sent" json:"is_sent"`
	SentAt   *time.Time `db:"sent_at" json:"sent_at,omitempty"`
}

type TaskHistory struct {
	ID          uuid.UUID `db:"id" json:"id"`
	TaskID      uuid.UUID `db:"task_id" json:"task_id"`
	CompletedBy uuid.UUID `db:"completed_by" json:"completed_by"`
	CompletedAt time.Time `db:"completed_at" json:"completed_at"`
}

type CreateTaskRequest struct {
	HouseID          uuid.UUID        `json:"house_id"`
	CreatedBy        uuid.UUID        `json:"created_by"`
	AssignedTo       *uuid.UUID       `json:"assigned_to,omitempty"`
	Title            string           `json:"title"`
	TaskType         TaskType         `json:"task_type"`
	Priority         TaskPriority     `json:"priority"`
	ReminderStrategy ReminderStrategy `json:"reminder_strategy"`
	DueAt            *time.Time       `json:"due_at,omitempty"`
	IntervalDays     *int             `json:"interval_days,omitempty"`
}

type UpdateTaskRequest struct {
	AssignedTo       *uuid.UUID        `json:"assigned_to,omitempty"`
	Title            *string           `json:"title,omitempty"`
	Priority         *TaskPriority     `json:"priority,omitempty"`
	ReminderStrategy *ReminderStrategy `json:"reminder_strategy,omitempty"`
	DueAt            *time.Time        `json:"due_at,omitempty"`
	IntervalDays     *int              `json:"interval_days,omitempty"`
}

type CompleteTaskRequest struct {
	CompletedBy uuid.UUID `json:"completed_by"`
}
