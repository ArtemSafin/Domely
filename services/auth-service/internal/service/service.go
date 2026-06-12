package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/model"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/repository"
)

type AuthService struct {
	repo      *repository.AuthRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func New(repo *repository.AuthRepository, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) (*model.TokenResponse, error) {
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// создаём пользователя
	user, err := s.repo.CreateUser(ctx, req.TelegramID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// хешируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// сохраняем credentials
	if err := s.repo.CreateCredential(ctx, user.ID, string(hash)); err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	return s.generateToken(user.ID, user.TelegramID, user.Name)
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.TokenResponse, error) {
	user, err := s.repo.GetUserByTelegramID(ctx, req.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	cred, err := s.repo.GetCredentialByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("credentials not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return s.generateToken(user.ID, user.TelegramID, user.Name)
}

func (s *AuthService) ValidateToken(tokenStr string) (*model.Claims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	userID, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid user_id in token")
	}

	return &model.Claims{
		UserID:     userID,
		TelegramID: int64(claims["telegram_id"].(float64)),
		Name:       claims["name"].(string),
	}, nil
}

func (s *AuthService) generateToken(userID uuid.UUID, telegramID int64, name string) (*model.TokenResponse, error) {
	expiresAt := time.Now().Add(s.jwtExpiry)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     userID.String(),
		"telegram_id": telegramID,
		"name":        name,
		"exp":         expiresAt.Unix(),
	})

	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &model.TokenResponse{
		AccessToken: signed,
		ExpiresAt:   expiresAt,
		UserID:      userID,
		Name:        name,
	}, nil
}