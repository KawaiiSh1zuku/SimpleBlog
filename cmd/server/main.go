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
		utils.LogError(context.Background(), "server stopped", err)
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

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.ConnectionString())
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}
	poolConfig.ConnConfig.Tracer = repository.NewSlowQueryTracer(time.Duration(cfg.Database.SlowQueryMS) * time.Millisecond)
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
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
	linkRepository := repository.NewLinkRepository(pool)
	feedRepository := repository.NewFeedRepository(pool)
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
	linkService := service.NewLinkService(linkRepository)
	rssService := service.NewRSSService(feedRepository, service.RSSSite{Name: cfg.Site.Name, Description: cfg.Site.Description, BaseURL: cfg.Site.BaseURL})
	uploadService := service.NewUploadService(uploadSigner, cfg.S3.PublicBaseURL)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	postHandler := handler.NewPostHandler(postService)
	publicHandler := handler.NewPublicHandler(publicService)
	commentHandler := handler.NewCommentHandler(commentService, cfg.Gravatar.BaseURL)
	uploadHandler := handler.NewUploadHandler(uploadService)
	linkHandler := handler.NewLinkHandler(linkService)
	rssHandler := handler.NewRSSHandler(rssService)

	turnstileService := service.NewTurnstileService(cfg.Turnstile.Secret)
	serviceHandler := handler.NewServiceHandler()

	serviceRoutes := router.ServiceRouter(serviceHandler, turnstileService)
	authRoutes := router.AuthRouter(authHandler, tokenAuth, userRepository)
	adminRoutes := router.AdminRouter(userHandler, categoryHandler, postHandler, commentHandler, uploadHandler, linkHandler, tokenAuth, userRepository)
	rssRoutes := router.RSSRouter(rssHandler)
	publicRoutes := router.PublicRouter(publicHandler, commentHandler, linkHandler, turnstileService)
	rootRouter := router.New(serviceRoutes, authRoutes, adminRoutes, rssRoutes, publicRoutes, cfg.Server.TrustProxyHeaders)

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
