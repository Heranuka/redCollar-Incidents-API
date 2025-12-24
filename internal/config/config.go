package config

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env      string         `json:"env"`
	Http     HttpConfig     `json:"http"`
	Postgres PostgresConfig `json:"postgres"`
	Redis    RedisConfig    `json:"redis"`
	APIKey   string         `json:"api_key,omitempty"`
	Webhook  WebhookConfig  `json:"webhook"`
}

type HttpConfig struct {
	Port            string        `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	SSLMode  string `json:"ssl_mode"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db"`
}

type WebhookConfig struct {
	URL      string `json:"url"`
	Disabled bool   `json:"disabled"` // ← НОВОЕ: выключатель webhook
}

func Load(ctx context.Context, logger *slog.Logger) (*Config, error) {
	// Загружаем .env (не обязательно)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Warn(".env load warning", slog.Any("error", err))
	}

	cfg := &Config{
		Env: getEnv(logger, "ENV", "local"),
		Http: HttpConfig{
			Port:            getEnv(logger, "HTTP_PORT", "8080"),
			ReadTimeout:     getEnvDuration(logger, "HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvDuration(logger, "HTTP_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getEnvDuration(logger, "HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Postgres: PostgresConfig{
			Host:     getEnv(logger, "POSTGRES_HOST", "pg-local"),
			Port:     getEnvInt(logger, "POSTGRES_PORT", 5432),
			Database: getEnv(logger, "POSTGRES_DB", "redcollar_db"), // ✅ ИСПРАВЛЕНО!
			User:     getEnv(logger, "POSTGRES_USER", "postgres"),
			Password: getEnv(logger, "POSTGRES_PASSWORD", "postgres"),
			SSLMode:  getEnv(logger, "POSTGRES_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv(logger, "REDIS_ADDR", "redis-local:6379"), // ✅ ИСПРАВЛЕНО!
			Password: getEnv(logger, "REDIS_PASSWORD", ""),
			DB:       getEnvInt(logger, "REDIS_DB", 0),
		},
		APIKey: getEnv(logger, "API_KEY", "super-secret-key"),
		Webhook: WebhookConfig{
			URL:      getEnv(logger, "WEBHOOK_URL", "https://webhook.site/5fc9c082-7cf6-47c7-94b5-be7d570346d1"),
			Disabled: getEnvBool(logger, "WEBHOOK_DISABLED", false),
		},
	}

	// ✅ ВАЛИДАЦИЯ
	if err := cfg.Validate(ctx, logger); err != nil {
		return nil, err
	}

	logger.Info("Config loaded successfully",
		slog.String("env", cfg.Env),
		slog.String("http_port", cfg.Http.Port),
		slog.String("postgres_db", cfg.Postgres.Database),
		slog.String("redis_addr", cfg.Redis.Addr),
		slog.String("webhook_url", cfg.Webhook.URL),
		slog.Bool("webhook_disabled", cfg.Webhook.Disabled))

	return cfg, nil
}

func (c *Config) Validate(ctx context.Context, logger *slog.Logger) error {
	if c.Http.Port == "" || !strings.HasPrefix(c.Http.Port, ":") && c.Http.Port[0] != ':' {
		return errors.New("HTTP_PORT must be like ':8080'")
	}

	if c.Postgres.Host == "" {
		return errors.New("POSTGRES_HOST required")
	}

	if c.Webhook.Disabled {
		logger.Warn("Webhooks DISABLED via WEBHOOK_DISABLED=true")
	}

	return nil
}

// Helpers с логами
func getEnv(logger *slog.Logger, key, def string) string {
	if v := os.Getenv(key); v != "" {
		logger.Debug("env found", slog.String("key", key), slog.String("value", v))
		return v
	}
	logger.Debug("env default", slog.String("key", key), slog.String("value", def))
	return def
}

func getEnvInt(logger *slog.Logger, key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			logger.Debug("env int found", slog.String("key", key), slog.Int("value", n))
			return n
		}
	}
	logger.Debug("env int default", slog.String("key", key), slog.Int("value", def))
	return def
}

func getEnvDuration(logger *slog.Logger, key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			logger.Debug("env duration found", slog.String("key", key), slog.Duration("value", d))
			return d
		}
	}
	logger.Debug("env duration default", slog.String("key", key), slog.Duration("value", def))
	return def
}

func getEnvBool(logger *slog.Logger, key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			logger.Debug("env bool found", slog.String("key", key), slog.Bool("value", b))
			return b
		}
	}
	logger.Debug("env bool default", slog.String("key", key), slog.Bool("value", def))
	return def
}
