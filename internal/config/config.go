package config

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"strconv"
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

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db"`
}

type WebhookConfig struct {
	URL      string `json:"url"`
	Disabled bool   `json:"disabled"`
}

func Load(ctx context.Context) (*Config, error) {

	stdLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		stdLogger.Warn(".env load warning", slog.Any("error", err))
	}

	cfg := &Config{
		Env: getEnv("ENV", "local"),
		Http: HttpConfig{
			Port:            getEnv("HTTP_PORT", ":8080"),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "pg-local"),
			Port:            getEnvInt("POSTGRES_PORT", 5432),
			Database:        getEnv("POSTGRES_DB", "redcollar_db"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			SSLMode:         getEnv("POSTGRES_SSL_MODE", "disable"),
			MaxConns:        20,
			MinConns:        1,
			MaxConnLifetime: 1 * time.Hour,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis-local:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		APIKey: getEnv("API_KEY", "super-secret-key"),
		Webhook: WebhookConfig{
			URL:      getEnv("WEBHOOK_URL", "https://webhook.site/5fc9c082-7cf6-47c7-94b5-be7d570346d1"),
			Disabled: getEnvBool("WEBHOOK_DISABLED", false),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	stdLogger.Info("Config loaded successfully",
		slog.String("env", cfg.Env),
		slog.String("http_port", cfg.Http.Port),
		slog.String("postgres_db", cfg.Postgres.Database),
		slog.String("redis_addr", cfg.Redis.Addr),
		slog.String("webhook_url", cfg.Webhook.URL))

	return cfg, nil
}

func (c *Config) Validate() error {

	if c.Http.Port == "" || (len(c.Http.Port) > 0 && c.Http.Port[0] != ':') {
		return errors.New("HTTP_PORT must start with ':' like ':8080'")
	}

	if c.Postgres.Host == "" {
		return errors.New("POSTGRES_HOST required")
	}

	if c.Webhook.Disabled {
		log.Println("WARN: Webhooks DISABLED via WEBHOOK_DISABLED=true")
	}

	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
