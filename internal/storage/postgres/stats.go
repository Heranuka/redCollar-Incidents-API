package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"redCollar/internal/domain"
	"redCollar/pkg/e"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewStats(pool *pgxpool.Pool, logger *slog.Logger) *StatsRepo {
	return &StatsRepo{pool: pool, logger: logger}
}

func (p *StatsRepo) SaveCheck(ctx context.Context, check *domain.LocationCheck) error {
	const op = "postgres.LocationCheck.Save"

	if check == nil || check.UserID.String() == "" {
		return fmt.Errorf("%s: %w", op, e.ErrInvalidInput)
	}

	const query = `
		INSERT INTO location_checks (id, user_id, lat, lng, incident_ids, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if check.ID == uuid.Nil {
		check.ID = uuid.New()
	}
	if check.CheckedAt.IsZero() {
		check.CheckedAt = time.Now().UTC()
	}

	_, err := p.pool.Exec(ctx, query,
		check.ID,
		check.UserID,
		check.Lat,
		check.Lng,
		check.IncidentIDs,
		check.CheckedAt,
	)
	if err != nil {
		p.logger.Error("db exec failed", slog.String("op", op), slog.Any("error", err))
		return e.WrapError(ctx, op, err)
	}

	return nil
}

func (p *StatsRepo) CountUniqueUsers(ctx context.Context, minutes int) (int64, error) {
	const op = "postgres.LocationCheck.CountUniqueUsers"

	if minutes <= 0 || minutes > 1440 {
		return 0, fmt.Errorf("%s: %w", op, e.ErrInvalidInput)
	}

	// ✅ безопасная параметризация интервала: число * interval '1 minute' [web:433]
	const query = `
		SELECT COUNT(DISTINCT user_id)
		FROM location_checks
		WHERE checked_at >= NOW() - ($1 * INTERVAL '1 minute')
	`

	var cnt int64
	if err := p.pool.QueryRow(ctx, query, minutes).Scan(&cnt); err != nil { // ошибка придёт на Scan [web:430]
		p.logger.Error("db queryrow scan failed",
			slog.String("op", op),
			slog.Any("error", err),
			slog.Int("minutes", minutes),
		)
		return 0, e.WrapError(ctx, op, err)
	}

	return cnt, nil
}
