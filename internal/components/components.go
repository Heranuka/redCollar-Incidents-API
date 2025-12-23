package components

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"redCollar/internal/api"
	"redCollar/internal/config"
	"redCollar/internal/service"
	"redCollar/internal/storage/postgres"
	"redCollar/internal/storage/redis"
	"redCollar/pkg/logger"
	"time"
)

type Components struct {
	logger     *slog.Logger
	HttpServer *api.Server
	Postgres   *postgres.Postgres
	Redis      *redis.Redis        // ← ДОБАВЬ Redis
	WebhookQ   *redis.WebhookQueue // ← ДОБАВЬ очередь
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
	redisClient, err := redis.NewRedis(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init redis: %w", err)
	}

	webhookQueue := redis.NewWebhookQueue(redisClient.Client, "webhooks:queue")

	// ← ИСПРАВЬ: убери дублирующий NewService()
	adminSvc := service.NewAdminIncidentService(storage.AdminIncidents())
	publicSvc := service.NewPublicIncidentService(storage.PublicIncidents(), webhookQueue, logger, 1.0)
	statsSvc := service.NewStatsService(storage.Stats())

	srv := service.NewService(adminSvc, publicSvc, statsSvc) // ← ОДИН раз

	httpServer := api.NewServer(cfg, logger, srv)
	logger.Info("Initialized server")

	return &Components{
		logger:     logger,
		HttpServer: httpServer,
		Postgres:   storage,
		Redis:      redisClient,  // ← ДОБАВЬ
		WebhookQ:   webhookQueue, // ← ДОБАВЬ
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
	c.logger.Info("Завершение работы компонентов началось")

	c.Postgres.Pool.Close()
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			c.logger.Error("Redis close failed", slog.String("err", err.Error()))
		}
	}

	c.logger.Info("Все компоненты успешно завершили работу",
		slog.Duration("latency", time.Since(start)))
}
