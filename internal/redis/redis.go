package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"redCollar/internal/config"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to ping Redis", slog.String("error", err.Error()))
		if err := rdb.Close(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	logger.Info("Connected to Redis successfully")

	return &Redis{Client: rdb}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
