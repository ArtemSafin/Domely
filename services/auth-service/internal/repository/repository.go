package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/model"
)

type AuthRepository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

type User struct {
	ID         uuid.UUID `db:"id"`
	TelegramID int64     `db:"telegram_id"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

func (r *AuthRepository) CreateUser(ctx context.Context, telegramID int64, name string) (*User, error) {
	u := &User{
		ID:         uuid.New(),
		TelegramID: telegramID,
		Name:       name,
		CreatedAt:  time.Now(),
	}
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO users (id, telegram_id, name, created_at)
		VALUES (:id, :telegram_id, :name, :created_at)
		ON CONFLICT (telegram_id) DO NOTHING`, u)
	if err != nil {
		return nil, err
	}
	return r.GetUserByTelegramID(ctx, telegramID)
}

func (r *AuthRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE telegram_id = $1`, telegramID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepository) CreateCredential(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO credentials (id, user_id, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET password_hash = $3`,
		uuid.New(), userID, passwordHash, time.Now())
	return err
}

func (r *AuthRepository) GetCredentialByUserID(ctx context.Context, userID uuid.UUID) (*model.Credential, error) {
	var c model.Credential
	err := r.db.GetContext(ctx, &c,
		`SELECT * FROM credentials WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return &c, nil
}