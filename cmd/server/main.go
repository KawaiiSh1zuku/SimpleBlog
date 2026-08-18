package main

import (
	"context"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/router"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	// ------------------- 初始化 APP -------------------
	w := os.Stdout
	var logLevel slog.LevelVar

	// 先设置默认级别（debug）
	logLevel.Set(slog.LevelDebug)

	// 创建并直接设置为全局默认 Logger
	slog.SetDefault(slog.New(
		tint.NewTextHandler(w, &tint.Options{
			Level:      &logLevel,
			TimeFormat: time.DateTime,
		}),
	))

	// 加载配置（此时如果 LoadConfig 里用了 slog，它会自动走全局配置）
	cfg, err := config.LoadConfig()
	if err != nil {
		// 这里直接用 slog.Error（全局）
		slog.Error("Failed to load config", "error", err)
		os.Exit(1) // 加载配置失败直接退出，避免空指针
	}

	// 根据配置文件更新日志级别
	logLevel.Set(slog.Level(utils.MapLogLevel(cfg.App.Loglevel)))

	// ------------------- 初始化数据库连接池 -------------------
	ctx := context.Background()
	databaseURL := "postgres://" + cfg.Database.Username + ":" + cfg.Database.Password + "@" + cfg.Database.Host + ":" + cfg.Database.Port + "/" + cfg.Database.DBName
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
	}
	defer pool.Close()

	// ------------------- 初始化 HTTP -------------------

	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	if cfg.App.Debug {
		r.Use(middleware.ClientIPFromRemoteAddr)
	} else {
		r.Use(middleware.ClientIPFromHeader("X-Real-IP"))
	}
	r.Use(middleware.Recoverer) // 捕获 panic，避免服务崩溃

	// 初始化 Handlers
	tokenAuth := jwtauth.New("HS256", cfg.JWT.Secret, nil)
	authRepository := repository.NewAuthRepository(pool)
	authService := service.NewAuthService(authRepository, tokenAuth)
	adminHandler := handler.NewAdminHandler(authService)

	turnstileService := service.NewTurnstileService(cfg.Turnstile.Secret)
	serviceHandler := handler.NewServiceHandler()

	// 挂载模块
	r.Mount("/service", router.ServiceRouter(serviceHandler, turnstileService))
	r.Mount("/admin", router.AdminRouter(adminHandler))

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	slog.Info("Starting server", "address", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		return
	}
}
