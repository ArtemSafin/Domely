package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ArtemSafin/Domely/services/task-service/internal/model"
)

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) CreateUser(ctx context.Context, u *model.User) error {
	query := `INSERT INTO users (id, telegram_id, name, created_at)
		VALUES (:id, :telegram_id, :name, :created_at)
		ON CONFLICT (telegram_id) DO NOTHING`
	_, err := r.db.NamedExecContext(ctx, query, u)
	return err
}

func (r *TaskRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE telegram_id = $1`, telegramID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *TaskRepository) CreateHouse(ctx context.Context, h *model.House) error {
	query := `INSERT INTO houses (id, name, owner_id, created_at)
		VALUES (:id, :name, :owner_id, :created_at)`
	_, err := r.db.NamedExecContext(ctx, query, h)
	return err
}

func (r *TaskRepository) GetHousesByUserID(ctx context.Context, userID uuid.UUID) ([]model.House, error) {
	var houses []model.House
	query := `SELECT h.* FROM houses h
		JOIN house_members hm ON hm.house_id = h.id
		WHERE hm.user_id = $1`
	err := r.db.SelectContext(ctx, &houses, query, userID)
	return houses, err
}

func (r *TaskRepository) AddHouseMember(ctx context.Context, m *model.HouseMember) error {
	query := `INSERT INTO house_members (house_id, user_id, role)
		VALUES (:house_id, :user_id, :role)
		ON CONFLICT DO NOTHING`
	_, err := r.db.NamedExecContext(ctx, query, m)
	return err
}

func (r *TaskRepository) GetHouseMembers(ctx context.Context, houseID uuid.UUID) ([]model.User, error) {
	var users []model.User
	query := `SELECT u.* FROM users u
		JOIN house_members hm ON hm.user_id = u.id
		WHERE hm.house_id = $1`
	err := r.db.SelectContext(ctx, &users, query, houseID)
	return users, err
}

func (r *TaskRepository) IsHouseMember(ctx context.Context, houseID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND user_id = $2`,
		houseID, userID)
	return count > 0, err
}

func (r *TaskRepository) CreateTask(ctx context.Context, t *model.Task) error {
	query := `INSERT INTO tasks (
			id, house_id, created_by, assigned_to, title,
			task_type, priority, reminder_strategy,
			due_at, next_run_at, interval_days, is_active, created_at
		) VALUES (
			:id, :house_id, :created_by, :assigned_to, :title,
			:task_type, :priority, :reminder_strategy,
			:due_at, :next_run_at, :interval_days, :is_active, :created_at
		)`
	_, err := r.db.NamedExecContext(ctx, query, t)
	return err
}

func (r *TaskRepository) GetTaskByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var t model.Task
	err := r.db.GetContext(ctx, &t, `SELECT * FROM tasks WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepository) GetTasksByHouseID(ctx context.Context, houseID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.SelectContext(ctx, &tasks,
		`SELECT * FROM tasks WHERE house_id = $1 AND is_active = TRUE ORDER BY created_at DESC`,
		houseID)
	return tasks, err
}

func (r *TaskRepository) GetTasksByAssignedTo(ctx context.Context, userID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.SelectContext(ctx, &tasks,
		`SELECT * FROM tasks WHERE assigned_to = $1 AND is_active = TRUE ORDER BY created_at DESC`,
		userID)
	return tasks, err
}

func (r *TaskRepository) UpdateTask(ctx context.Context, id uuid.UUID, req *model.UpdateTaskRequest) error {
	query := `UPDATE tasks SET`
	args := []interface{}{}
	argIdx := 1

	if req.Title != nil {
		query += fmt.Sprintf(" title = $%d,", argIdx)
		args = append(args, *req.Title)
		argIdx++
	}
	if req.AssignedTo != nil {
		query += fmt.Sprintf(" assigned_to = $%d,", argIdx)
		args = append(args, *req.AssignedTo)
		argIdx++
	}
	if req.Priority != nil {
		query += fmt.Sprintf(" priority = $%d,", argIdx)
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.ReminderStrategy != nil {
		query += fmt.Sprintf(" reminder_strategy = $%d,", argIdx)
		args = append(args, *req.ReminderStrategy)
		argIdx++
	}
	if req.DueAt != nil {
		query += fmt.Sprintf(" due_at = $%d,", argIdx)
		args = append(args, *req.DueAt)
		argIdx++
	}
	if req.IntervalDays != nil {
		query += fmt.Sprintf(" interval_days = $%d,", argIdx)
		args = append(args, *req.IntervalDays)
		argIdx++
	}

	query = query[:len(query)-1]
	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *TaskRepository) DeleteTask(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET is_active = FALSE WHERE id = $1`, id)
	return err
}

func (r *TaskRepository) CompleteTask(ctx context.Context, taskID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_history (id, task_id, completed_by, completed_at) VALUES ($1, $2, $3, $4)`,
		uuid.New(), taskID, userID, time.Now())
	if err != nil {
		return err
	}

	var t model.Task
	err = tx.GetContext(ctx, &t, `SELECT * FROM tasks WHERE id = $1`, taskID)
	if err != nil {
		return err
	}

	if t.TaskType == model.TaskTypeRecurring && t.IntervalDays != nil {
		nextRun := time.Now().AddDate(0, 0, *t.IntervalDays)
		_, err = tx.ExecContext(ctx,
			`UPDATE tasks SET next_run_at = $1 WHERE id = $2`, nextRun, taskID)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE tasks SET is_active = FALSE WHERE id = $1`, taskID)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *TaskRepository) GetTaskHistory(ctx context.Context, houseID uuid.UUID) ([]model.TaskHistory, error) {
	var history []model.TaskHistory
	query := `SELECT th.* FROM task_history th
		JOIN tasks t ON t.id = th.task_id
		WHERE t.house_id = $1
		ORDER BY th.completed_at DESC
		LIMIT 50`
	err := r.db.SelectContext(ctx, &history, query, houseID)
	return history, err
}

func (r *TaskRepository) CreateReminderRules(ctx context.Context, rules []model.ReminderRule) error {
	query := `INSERT INTO reminder_rules (id, task_id, remind_at, is_sent)
		VALUES (:id, :task_id, :remind_at, :is_sent)`
	_, err := r.db.NamedExecContext(ctx, query, rules)
	return err
}

func (r *TaskRepository) GetPendingReminders(ctx context.Context) ([]model.ReminderRule, error) {
	var rules []model.ReminderRule
	err := r.db.SelectContext(ctx, &rules,
		`SELECT * FROM reminder_rules WHERE remind_at <= NOW() AND is_sent = FALSE`)
	return rules, err
}

func (r *TaskRepository) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reminder_rules SET is_sent = TRUE, sent_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *TaskRepository) DeleteReminderRulesByTaskID(ctx context.Context, taskID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM reminder_rules WHERE task_id = $1 AND is_sent = FALSE`, taskID)
	return err
}
