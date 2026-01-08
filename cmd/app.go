package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"redCollar/internal/components"
	"redCollar/internal/config"
	"sync"
	"syscall"
)

func Run() error {
	logger := components.SetupLogger("local")

	// ✅ Загружаем config ПЕРЕД компонентами
	cfg, err := config.Load(context.Background())
	if err != nil {
		logger.Error("load config failed", slog.Any("err", err))
		return err
	}

	logger.Info("Config OK",
		slog.String("postgres_db", cfg.Postgres.Database),
		slog.String("redis_addr", cfg.Redis.Addr),
		slog.String("http_port", cfg.Http.Port))

	if cfg.APIKey == "" {
		return fmt.Errorf("API_KEY is empty")
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comps, err := components.InitComponents(appCtx, cfg, logger)
	if err != nil {
		logger.Error("could not init components", slog.Any("err", err))
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := comps.HttpServer.Run(ctx); err != nil {
			logger.Error("http server failed", slog.Any("err", err))
		}
		logger.Info("http server stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("🚀 webhookSender goroutine launched")
		comps.WebhookSender.Run(ctx)
	}()

	// 3. Запуск Воркера 2
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("🚀 locationChecker goroutine launched")
		comps.LocationChecker.Run(ctx)
	}()
	// Graceful shutdown
	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
	<-quitChan

	logger.Info("captured signal, initiating shutdown")
	stop()
	wg.Wait()

	comps.ShutdownAll()
	logger.Info("gracefully shut down")

	return nil
}
