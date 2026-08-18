package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/router"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/storage"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/utils"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	logLevel := configureLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logLevel.Set(slog.Level(utils.MapLogLevel(cfg.App.Loglevel)))

	pool, err := pgxpool.New(ctx, cfg.Database.ConnectionString())
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	tokenAuth := jwtauth.New("HS256", cfg.JWT.Secret, nil)
	userRepository := repository.NewUserRepository(pool)
	categoryRepository := repository.NewCategoryRepository(pool)
	postRepository := repository.NewPostRepository(pool)
	commentRepository := repository.NewCommentRepository(pool)
	uploadSigner, err := storage.NewS3Signer(cfg.S3)
	if err != nil {
		return err
	}
	authService := service.NewAuthService(userRepository, tokenAuth)
	userService := service.NewUserService(userRepository)
	categoryService := service.NewCategoryService(categoryRepository)
	postService := service.NewPostService(postRepository)
	publicService := service.NewPublicService(postRepository, categoryRepository)
	commentService := service.NewCommentService(commentRepository)
	uploadService := service.NewUploadService(uploadSigner, cfg.S3.PublicBaseURL)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	postHandler := handler.NewPostHandler(postService)
	publicHandler := handler.NewPublicHandler(publicService)
	commentHandler := handler.NewCommentHandler(commentService)
	uploadHandler := handler.NewUploadHandler(uploadService)

	turnstileService := service.NewTurnstileService(cfg.Turnstile.Secret)
	serviceHandler := handler.NewServiceHandler()

	serviceRoutes := router.ServiceRouter(serviceHandler, turnstileService)
	authRoutes := router.AuthRouter(authHandler, tokenAuth, userRepository)
	adminRoutes := router.AdminRouter(userHandler, categoryHandler, postHandler, commentHandler, uploadHandler, tokenAuth, userRepository)
	publicRoutes := router.PublicRouter(publicHandler, commentHandler, turnstileService)
	rootRouter := router.New(serviceRoutes, authRoutes, adminRoutes, publicRoutes, cfg.App.Debug)

	server := newHTTPServer(cfg.Server.Address(), rootRouter)
	slog.Info("starting server", "address", server.Addr)
	return serve(ctx, server)
}

func configureLogger() *slog.LevelVar {
	var logLevel slog.LevelVar
	logLevel.Set(slog.LevelDebug)
	slog.SetDefault(slog.New(
		tint.NewTextHandler(os.Stdout, &tint.Options{
			Level:      &logLevel,
			TimeFormat: time.DateTime,
		}),
	))
	return &logLevel
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serve(ctx context.Context, server *http.Server) error {
	serverErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return <-serverErr
	}
}
