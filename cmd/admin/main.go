package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap admin:", err)
		os.Exit(1)
	}
}

func run() error {
	username := flag.String("username", "admin", "admin username")
	nickname := flag.String("nickname", "Administrator", "admin nickname")
	avatarURL := flag.String("avatar-url", "", "optional avatar URL")
	flag.Parse()

	password := os.Getenv("ADMIN_PASSWORD")
	if strings.TrimSpace(password) == "" {
		return errors.New("ADMIN_PASSWORD is required")
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.Database.ConnectionString())
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	users := service.NewUserService(repository.NewUserRepository(pool))
	user, err := users.BootstrapAdmin(ctx, service.CreateUserInput{
		Username: *username, Password: password, Nickname: *nickname, AvatarURL: *avatarURL,
	})
	if err != nil {
		return err
	}
	fmt.Printf("created admin %s (%s)\n", user.Username, user.ID)
	return nil
}
