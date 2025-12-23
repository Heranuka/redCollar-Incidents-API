package postgres

import (
	"context"
	"redCollar/internal/domain"
	"redCollar/pkg/e"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepo struct{ pool *pgxpool.Pool }

func NewStats(pool *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{pool: pool}
}

func (p StatsRepo) SaveCheck(ctx context.Context, check *domain.LocationCheck) error {
	const op = "postgres.LocationCheck.Save"

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
		check.IncidentIDs, // pgx сам умеет uuid[]
		check.CheckedAt,
	)
	return e.WrapError(ctx, op, err)
}

func (p StatsRepo) CountUniqueUsers(ctx context.Context, minutes int) (int64, error) {
	const op = "postgres.LocationCheck.CountUniqueUsers"

	const query = `
		SELECT COUNT(DISTINCT user_id)
		FROM location_checks
		WHERE checked_at >= NOW() - ($1 || ' minutes')::interval
	`

	var cnt int64
	err := p.pool.QueryRow(ctx, query, minutes).Scan(&cnt)
	if err != nil {
		return 0, e.WrapError(ctx, op, err)
	}
	return cnt, nil
}
