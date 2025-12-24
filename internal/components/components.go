package components

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"redCollar/internal/api"
	"redCollar/internal/config"
	redis2 "redCollar/internal/redis"
	"redCollar/internal/service"
	"redCollar/internal/storage/postgres"
	"redCollar/internal/workers"
	"redCollar/pkg/logger"
	"time"
)

type Components struct {
	logger          *slog.Logger
	HttpServer      *api.Server
	Postgres        *postgres.Postgres
	Redis           *redis2.Redis
	WebhookQ        *redis2.WebhookQueue
	LocationChecker *workers.LocationChecker
	webhookSender   *service.WebhookSender // ← ДОБАВИЛИ!
}

func InitComponents(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Components, error) {
	logger.Info("Initializing Postgres")

	storage, err := postgres.NewPostgres(ctx, cfg, logger)
	if err != nil {
		logger.Error("Failed to init postgres",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	logger.Info("Initializing Redis")
	redisClient, err := redis2.NewRedis(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init redis: %w", err)
	}

	webhookQueue := redis2.NewWebhookQueue(redisClient.Client, "webhooks:queue")
	webhookSender := service.NewWebhookSender(logger, cfg.Webhook, webhookQueue)

	logger.Info("🔥 Starting webhookSender",
		slog.String("url", cfg.Webhook.URL),
		slog.String("queue", "webhooks:queue")) // ← ИСПРАВИЛИ!

	go func() {
		logger.Info("🚀 webhookSender goroutine launched")
		webhookSender.Run(ctx)
	}()

	cache := redis2.NewIncidentCache(redisClient)
	adminSvc := service.NewAdminIncidentService(storage.AdminIncidents(), cache)
	statsRepo := storage.Stats() // *postgres.StatsRepo
	publicSvc := service.NewPublicIncidentService(cache, statsRepo, webhookQueue, logger, 1.0)
	statsSvc := service.NewStatsService(storage.Stats())
	locationChecker := workers.NewLocationChecker(cache, 10)

	logger.Info("🚀 Starting locationChecker")
	locationChecker.Start(ctx)

	srv := service.NewService(adminSvc, publicSvc, statsSvc)

	httpServer := api.NewServer(cfg, logger, srv)
	logger.Info("Initialized server")

	return &Components{
		logger:          logger,
		HttpServer:      httpServer,
		Postgres:        storage,
		Redis:           redisClient,
		WebhookQ:        webhookQueue,
		LocationChecker: locationChecker,
		webhookSender:   webhookSender, // ← СОХРАНИЛИ!
	}, nil
}

func SetupLogger(env string) *slog.Logger {
	switch env {
	case "local":
		return logger.SetupPrettySlog()
	case "dev":
		return slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		)
	case "prod":
		return slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		)
	default:
		return slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		)
	}
}

func (c *Components) ShutdownAll() {
	start := time.Now()
	c.logger.Info("🛑 Завершение работы компонентов началось")

	// ✅ Останавливаем locationChecker
	if c.LocationChecker != nil {
		c.logger.Info("🛑 Stopping locationChecker...")
		c.LocationChecker.Stop()
	}

	// DB connections
	c.logger.Info("🛑 Closing Postgres...")
	c.Postgres.Pool.Close()

	c.logger.Info("🛑 Closing Redis...")
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			c.logger.Error("Redis close failed", slog.String("err", err.Error()))
		}
	}

	c.logger.Info("✅ Все компоненты завершили работу",
		slog.Duration("latency", time.Since(start)))
}
