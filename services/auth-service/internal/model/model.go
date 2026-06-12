package model

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type RegisterRequest struct {
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Password   string `json:"password"`
}

type LoginRequest struct {
	TelegramID int64  `json:"telegram_id"`
	Password   string `json:"password"`
}

type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
}

type Claims struct {
	UserID     uuid.UUID `json:"user_id"`
	TelegramID int64     `json:"telegram_id"`
	Name       string    `json:"name"`
}