package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"redCollar/internal/domain"
	"redCollar/pkg/e"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentPublic struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewIncidentPublic(pool *pgxpool.Pool, logger *slog.Logger) *IncidentPublic {
	return &IncidentPublic{pool: pool, logger: logger}
}

func (p *IncidentPublic) FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]uuid.UUID, error) {
	const op = "postgres.Incident.FindNearby"

	// минимальная защита от мусора
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || radiusKm <= 0 {
		return nil, fmt.Errorf("%s: %w", op, e.ErrInvalidInput)
	}

	// Важно: geo_point (geometry, 4326) -> distance в градусах.
	// Кастим к geography, чтобы расстояние было в метрах.
	const query = `
SELECT id
FROM incidents
WHERE status = 'active'
  AND ST_DWithin(
    geo_point::geography,
    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
    $3 * 1000
  )
`

	rows, err := p.pool.Query(ctx, query, lng, lat, radiusKm)
	if err != nil {
		p.logger.Error("db query failed", slog.String("op", op), slog.Any("error", err))
		return nil, e.WrapError(ctx, op, err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			p.logger.Error("row scan failed", slog.String("op", op), slog.Any("error", err))
			return nil, e.WrapError(ctx, op, err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		p.logger.Error("rows err", slog.String("op", op), slog.Any("error", err))
		return nil, e.WrapError(ctx, op, err)
	}

	return ids, nil
}

func (p *IncidentPublic) SaveCheck(ctx context.Context, check *domain.LocationCheck) error {
	const op = "postgres.LocationCheck.Save"

	if check == nil {
		return fmt.Errorf("%s: %w", op, e.ErrInvalidInput)
	}
	if check.UserID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, e.ErrInvalidInput)
	}
	if check.Lat < -90 || check.Lat > 90 || check.Lng < -180 || check.Lng > 180 {
		return fmt.Errorf("%s: %w", op, e.ErrInvalidCoordinates)
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
		p.logger.Error("db exec failed",
			slog.String("op", op),
			slog.Any("error", err),
			slog.String("user_id", check.UserID.String()),
		)
		return e.WrapError(ctx, op, err)
	}

	return nil
}
