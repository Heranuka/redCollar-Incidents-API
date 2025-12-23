package postgres

import (
	"context"
	"fmt"

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
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Database,
		cfg.Postgres.SSLMode,
	)

	logger.Info("Connecting to Postgres", "dsn", dsn)

	configNew, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Error("Failed to parse pgx config", slog.String("error", err.Error()))
		return nil, e.Wrap("storage.pg.NewPostgres.ParseConfig", err)
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
		pool.Close() // ← ДОБАВЬ: закрой пул при ошибке
		return nil, e.Wrap("storage.pg.NewPostgres.Ping", err)
	}
	logger.Info("Connected to Postgres successfully")

	pg := &Postgres{
		Pool:          pool,
		IncidentAdmin: NewIncidentAdmin(pool),
		Geo:           NewIncidentPublic(pool),
		Stat:          NewStats(pool),
	}

	logger.Info("Postgres repositories created")
	return pg, nil
}
