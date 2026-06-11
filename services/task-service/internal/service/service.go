package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ArtemSafin/Domely/services/task-service/internal/model"
	"github.com/ArtemSafin/Domely/services/task-service/internal/repository"
)

type TaskService struct {
	repo *repository.TaskRepository
}

func NewTaskService(repo *repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) RegisterUser(ctx context.Context, telegramID int64, name string) (*model.User, error) {
	u := &model.User{
		ID:         uuid.New(),
		TelegramID: telegramID,
		Name:       name,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return s.repo.GetUserByTelegramID(ctx, telegramID)
}

func (s *TaskService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	return s.repo.GetUserByTelegramID(ctx, telegramID)
}

func (s *TaskService) CreateHouse(ctx context.Context, name string, ownerID uuid.UUID) (*model.House, error) {
	h := &model.House{
		ID:        uuid.New(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateHouse(ctx, h); err != nil {
		return nil, fmt.Errorf("create house: %w", err)
	}
	if err := s.repo.AddHouseMember(ctx, &model.HouseMember{
		HouseID: h.ID,
		UserID:  ownerID,
		Role:    "owner",
	}); err != nil {
		return nil, fmt.Errorf("add owner as member: %w", err)
	}
	return h, nil
}

func (s *TaskService) GetHousesByUserID(ctx context.Context, userID uuid.UUID) ([]model.House, error) {
	return s.repo.GetHousesByUserID(ctx, userID)
}

func (s *TaskService) InviteMember(ctx context.Context, houseID, userID uuid.UUID) error {
	return s.repo.AddHouseMember(ctx, &model.HouseMember{
		HouseID: houseID,
		UserID:  userID,
		Role:    "member",
	})
}

func (s *TaskService) GetHouseMembers(ctx context.Context, houseID uuid.UUID) ([]model.User, error) {
	return s.repo.GetHouseMembers(ctx, houseID)
}

func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}
	ok, err := s.repo.IsHouseMember(ctx, req.HouseID, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("user is not a member of this house")
	}
	if req.AssignedTo != nil {
		ok, err = s.repo.IsHouseMember(ctx, req.HouseID, *req.AssignedTo)
		if err != nil {
			return nil, fmt.Errorf("check assignee membership: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("assignee is not a member of this house")
		}
	}
	task := &model.Task{
		ID:               uuid.New(),
		HouseID:          req.HouseID,
		CreatedBy:        req.CreatedBy,
		AssignedTo:       req.AssignedTo,
		Title:            req.Title,
		TaskType:         req.TaskType,
		Priority:         req.Priority,
		ReminderStrategy: req.ReminderStrategy,
		DueAt:            req.DueAt,
		IntervalDays:     req.IntervalDays,
		IsActive:         true,
		CreatedAt:        time.Now(),
	}
	if task.TaskType == model.TaskTypeRecurring && task.DueAt != nil {
		task.NextRunAt = task.DueAt
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	rules := s.buildReminderRules(task)
	if len(rules) > 0 {
		if err := s.repo.CreateReminderRules(ctx, rules); err != nil {
			return nil, fmt.Errorf("create reminder rules: %w", err)
		}
	}
	return task, nil
}

func (s *TaskService) GetTasksByHouseID(ctx context.Context, houseID uuid.UUID) ([]model.Task, error) {
	return s.repo.GetTasksByHouseID(ctx, houseID)
}

func (s *TaskService) GetTasksByAssignedTo(ctx context.Context, userID uuid.UUID) ([]model.Task, error) {
	return s.repo.GetTasksByAssignedTo(ctx, userID)
}

func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, req *model.UpdateTaskRequest) (*model.Task, error) {
	if err := s.repo.UpdateTask(ctx, id, req); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.ReminderStrategy != nil || req.DueAt != nil {
		if err := s.repo.DeleteReminderRulesByTaskID(ctx, id); err != nil {
			return nil, fmt.Errorf("delete old reminder rules: %w", err)
		}
		rules := s.buildReminderRules(task)
		if len(rules) > 0 {
			if err := s.repo.CreateReminderRules(ctx, rules); err != nil {
				return nil, fmt.Errorf("create reminder rules: %w", err)
			}
		}
	}
	return task, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteReminderRulesByTaskID(ctx, id); err != nil {
		return fmt.Errorf("delete reminder rules: %w", err)
	}
	return s.repo.DeleteTask(ctx, id)
}

func (s *TaskService) CompleteTask(ctx context.Context, taskID, userID uuid.UUID) error {
	if err := s.repo.DeleteReminderRulesByTaskID(ctx, taskID); err != nil {
		return fmt.Errorf("delete reminder rules: %w", err)
	}
	if err := s.repo.CompleteTask(ctx, taskID, userID); err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.TaskType == model.TaskTypeRecurring && task.IsActive {
		rules := s.buildReminderRules(task)
		if len(rules) > 0 {
			if err := s.repo.CreateReminderRules(ctx, rules); err != nil {
				return fmt.Errorf("create next reminder rules: %w", err)
			}
		}
	}
	return nil
}

func (s *TaskService) GetTaskHistory(ctx context.Context, houseID uuid.UUID) ([]model.TaskHistory, error) {
	return s.repo.GetTaskHistory(ctx, houseID)
}

func (s *TaskService) GetPendingReminders(ctx context.Context) ([]model.ReminderRule, error) {
	return s.repo.GetPendingReminders(ctx)
}

func (s *TaskService) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkReminderSent(ctx, id)
}

func (s *TaskService) buildReminderRules(t *model.Task) []model.ReminderRule {
	var targetTime time.Time
	if t.TaskType == model.TaskTypeRecurring && t.NextRunAt != nil {
		targetTime = *t.NextRunAt
	} else if t.DueAt != nil {
		targetTime = *t.DueAt
	} else {
		return nil
	}

	var rules []model.ReminderRule
	switch t.ReminderStrategy {
	case model.ReminderSimple:
		rules = append(rules, model.ReminderRule{
			ID: uuid.New(), TaskID: t.ID, RemindAt: targetTime,
		})
	case model.ReminderAdvance:
		rules = append(rules,
			model.ReminderRule{ID: uuid.New(), TaskID: t.ID, RemindAt: targetTime.AddDate(0, 0, -1)},
			model.ReminderRule{ID: uuid.New(), TaskID: t.ID, RemindAt: targetTime},
		)
	case model.ReminderMeeting:
		startOfDay := time.Date(targetTime.Year(), targetTime.Month(), targetTime.Day(),
			9, 0, 0, 0, targetTime.Location())
		rules = append(rules,
			model.ReminderRule{ID: uuid.New(), TaskID: t.ID, RemindAt: startOfDay},
			model.ReminderRule{ID: uuid.New(), TaskID: t.ID, RemindAt: targetTime.Add(-30 * time.Minute)},
			model.ReminderRule{ID: uuid.New(), TaskID: t.ID, RemindAt: targetTime.Add(-5 * time.Minute)},
		)
	}
	return rules
}

func (s *TaskService) validateCreateRequest(req *model.CreateTaskRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.TaskType == model.TaskTypeRecurring {
		if req.IntervalDays == nil {
			return fmt.Errorf("interval_days is required for recurring tasks")
		}
		if req.DueAt == nil {
			return fmt.Errorf("due_at is required for recurring tasks")
		}
	}
	if req.TaskType == model.TaskTypeOneTime && req.DueAt == nil {
		return fmt.Errorf("due_at is required for one_time tasks")
	}
	return nil
}
