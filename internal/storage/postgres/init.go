package postgres

import (
	"context"
	"fmt"
	"net/url"

	"log/slog"
	"redCollar/internal/config"
	"redCollar/pkg/e"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool          *pgxpool.Pool
	IncidentAdmin IncidentRepository
	Stat          StatsRepository
	Geo           GeoRepository
}

func NewPostgres(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Postgres, error) {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Postgres.User, cfg.Postgres.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Postgres.Host, cfg.Postgres.Port),
		Path:   cfg.Postgres.Database,
	}
	q := dsn.Query()
	q.Set("sslmode", cfg.Postgres.SSLMode)
	dsn.RawQuery = q.Encode()
	logger.Info("Connecting to Postgres", "dsn", dsn)

	configNew, err := pgxpool.ParseConfig(dsn.String())
	if err != nil {
		logger.Error("Failed to parse pgx config", slog.String("error", err.Error()))
		return nil, e.Wrap("storage.pg.NewPostgres.ParseConfig", err)
	}

	if cfg.Postgres.MaxConns > 0 {
		configNew.MaxConns = cfg.Postgres.MaxConns
	}
	if cfg.Postgres.MinConns > 0 {
		configNew.MinConns = cfg.Postgres.MinConns
	}
	if cfg.Postgres.MaxConnLifetime > 0 {
		configNew.MaxConnLifetime = cfg.Postgres.MaxConnLifetime
	}

	pool, err := pgxpool.NewWithConfig(ctx, configNew)
	if err != nil {
		logger.Error("Failed to create pgx pool", slog.String("error", err.Error()))
		return nil, e.Wrap("storage.pg.NewPostgres.NewWithConfig", err)
	}

	logger.Info("Pinging Postgres database")
	err = pool.Ping(ctx)
	if err != nil {
		logger.Error("Failed to ping Postgres database", slog.String("error", err.Error()))
		pool.Close()
		return nil, e.Wrap("storage.pg.NewPostgres.Ping", err)
	}
	logger.Info("Connected to Postgres successfully")

	pg := &Postgres{
		Pool:          pool,
		IncidentAdmin: NewIncidentAdmin(pool, logger),
		Geo:           NewIncidentPublic(pool, logger),
		Stat:          NewStats(pool, logger),
	}

	logger.Info("Postgres repositories created")
	return pg, nil
}
