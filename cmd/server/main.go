package main

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	w := os.Stdout
	var logLevel slog.LevelVar
	logLevel.Set(slog.LevelDebug)

	// Create a new logger
	logger := slog.New(tint.NewTextHandler(w, nil))

	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewTextHandler(w, &tint.Options{
			Level:      &logLevel,
			TimeFormat: time.DateTime,
		}),
	))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
	}

	logLevel.Set(slog.Level(utils.MapLogLevel(cfg.App.Loglevel))) // 更新 logLevel，同步配置文件

	r := chi.NewRouter()
	r.Use(middleware.Recoverer) // 捕获 panic，避免服务崩溃

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	logger.Info("Starting server", "address", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		logger.Error("Failed to start HTTP server", "error", err)
		return
	}
}
