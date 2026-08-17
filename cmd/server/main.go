package main

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/handlers"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/utils"
	"github.com/KawaiiSh1zuku/SimpleBlog/routers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	// ------------------- 初始化 HTTP -------------------

	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP"))
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Recoverer) // 捕获 panic，避免服务崩溃

	// 初始化 Handlers
	serviceHandler := handlers.NewServiceHandler()

	// 挂载模块
	r.Mount("/service", routers.ServiceRouter(serviceHandler))

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	slog.Info("Starting server", "address", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		return
	}
}
