// Command server is the HIS outpatient backend entrypoint: it loads config,
// opens the database, wires the app container and serves the HTTP API with
// graceful shutdown (PLAN §2.1).
//
// @title                      东软云医院 HIS 门诊管理 API
// @version                    1.0
// @description                门诊全流程后端：挂号收费 / 门诊医生 / 检查 / 检验 / 处置 / 药房。
// @BasePath                   /api
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/app"
	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/database"
	"github.com/neu-software-practice/software-practice-backend/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	setupLogger(cfg.LogLevel)

	db, err := database.NewMySQL(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	engine := router.New(app.NewContainer(db, cfg).Deps())
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
