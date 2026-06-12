package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/config"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/handler"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/repository"
	"github.com/ArtemSafin/Domely/services/auth-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// создаём таблицу credentials если не существует
	if err := runMigration(db); err != nil {
		log.Fatalf("migration: %v", err)
	}

	repo := repository.New(db)

	jwtExpiry, err := time.ParseDuration(cfg.JWTExpiry)
	if err != nil {
		log.Fatalf("invalid JWT_EXPIRY: %v", err)
	}

	svc := service.New(repo, cfg.JWTSecret, jwtExpiry)
	h := handler.New(svc)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      h.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("auth-service listening on :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}

func runMigration(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS credentials (
			id           UUID PRIMARY KEY,
			user_id      UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			password_hash TEXT NOT NULL,
			created_at   TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}