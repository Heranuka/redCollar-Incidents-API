package postgres

import (
	"context"
	"redCollar/internal/domain"
	"redCollar/pkg/e"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentPublic struct{ pool *pgxpool.Pool }

func NewIncidentPublic(pool *pgxpool.Pool) *IncidentPublic {
	return &IncidentPublic{pool: pool}
}

func (p *IncidentPublic) FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]uuid.UUID, error) {
	const op = "postgres.Incident.FindNearby"

	const query = `
		SELECT id
		FROM incidents
		WHERE status = 'active'
		  AND ST_DWithin(
		        geo_point,
		        ST_SetSRID(ST_MakePoint($1, $2), 4326),
		        $3 * 1000
		      )
	`

	rows, err := p.pool.Query(ctx, query, lng, lat, radiusKm)
	if err != nil {
		return nil, e.WrapError(ctx, op, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, e.WrapError(ctx, op, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, e.WrapError(ctx, op, err)
	}

	return ids, nil
}

func (p IncidentPublic) SaveCheck(ctx context.Context, check *domain.LocationCheck) error {
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
