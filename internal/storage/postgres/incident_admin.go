package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"redCollar/internal/domain"
	"redCollar/pkg/e"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentAdmin struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewIncidentAdmin(pool *pgxpool.Pool, logger *slog.Logger) *IncidentAdmin {
	return &IncidentAdmin{pool: pool, logger: logger}
}

func (p *IncidentAdmin) Create(ctx context.Context, incident *domain.Incident) error {
	const op = "postgres.Incident.Create"

	query := `
		INSERT INTO incidents (id, geo_point, radius_km, status, created_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4, $5, $6)
	`

	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}
	if incident.CreatedAt.IsZero() {
		incident.CreatedAt = time.Now().UTC()
	}
	if incident.Status == "" {
		fmt.Println("WARNING: Incident Status is empty-----------")
		incident.Status = domain.IncidentActive
	}

	_, err := p.pool.Exec(ctx, query,
		incident.ID,
		incident.Lng,
		incident.Lat,
		incident.RadiusKM,
		incident.Status,
		incident.CreatedAt,
	)
	if err != nil {
		p.logger.Error("db exec failed",
			slog.String("op", op),
			slog.Any("error", err),
		)
		return e.WrapError(ctx, op, err)
	}

	return nil
}

func (p *IncidentAdmin) List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error) {
	const op = "postgres.Incident.List"

	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	const countQuery = `SELECT COUNT(*) FROM incidents`

	var total int64
	if err := p.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		p.logger.Error("db count failed", slog.String("op", op), slog.Any("error", err))
		return nil, 0, e.WrapError(ctx, op, err)
	}

	const listQuery = `
		SELECT id,
			   ST_Y(geo_point::geometry) AS lat,
			   ST_X(geo_point::geometry) AS lng,
			   radius_km,
			   status,
			   created_at
		FROM incidents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := p.pool.Query(ctx, listQuery, limit, offset)
	if err != nil {
		p.logger.Error("db query failed", slog.String("op", op), slog.Any("error", err))
		return nil, 0, e.WrapError(ctx, op, err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		var inc domain.Incident
		if err := rows.Scan(
			&inc.ID,
			&inc.Lat,
			&inc.Lng,
			&inc.RadiusKM,
			&inc.Status,
			&inc.CreatedAt,
		); err != nil {
			p.logger.Error("row scan failed", slog.String("op", op), slog.Any("error", err))
			return nil, 0, e.WrapError(ctx, op, err)
		}
		incidents = append(incidents, &inc)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error("rows err", slog.String("op", op), slog.Any("error", err))
		return nil, 0, e.WrapError(ctx, op, err)
	}

	return incidents, total, nil
}

func (p *IncidentAdmin) Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	const op = "postgres.Incident.Get"

	const query = `
		SELECT id,
			   ST_Y(geo_point::geometry) AS lat,
			   ST_X(geo_point::geometry) AS lng,
			   radius_km,
			   status,
			   created_at
		FROM incidents
		WHERE id = $1
	`

	var inc domain.Incident
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&inc.ID,
		&inc.Lat,
		&inc.Lng,
		&inc.RadiusKM,
		&inc.Status,
		&inc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // pgxpool QueryRow.Scan => ErrNoRows [web:403]
			return nil, fmt.Errorf("%s: %w", op, e.ErrNotFound)
		}
		p.logger.Error("db queryrow scan failed", slog.String("op", op), slog.Any("error", err), slog.String("id", id.String()))
		return nil, e.WrapError(ctx, op, err)
	}

	return &inc, nil
}

func (p *IncidentAdmin) Update(ctx context.Context, incident *domain.Incident) error {
	const op = "postgres.Incident.Update"

	const query = `
		UPDATE incidents
		SET geo_point = ST_SetSRID(ST_MakePoint($2, $3), 4326),
			radius_km = $4,
			status    = $5
		WHERE id = $1
	`

	cmd, err := p.pool.Exec(ctx, query,
		incident.ID,
		incident.Lng,
		incident.Lat,
		incident.RadiusKM,
		incident.Status,
	)
	if err != nil {
		p.logger.Error("db exec failed", slog.String("op", op), slog.Any("error", err), slog.String("id", incident.ID.String()))
		return e.WrapError(ctx, op, err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, e.ErrNotFound)
	}

	return nil
}

func (p *IncidentAdmin) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.Incident.Delete"

	const query = `
		UPDATE incidents
		SET status = 'inactive'
		WHERE id = $1 AND status = 'active'
	`

	cmd, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		p.logger.Error("db exec failed", slog.String("op", op), slog.Any("error", err), slog.String("id", id.String()))
		return e.WrapError(ctx, op, err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, e.ErrNotFound)
	}

	return nil
}

func (p *IncidentAdmin) ListActive(ctx context.Context) ([]*domain.Incident, error) {
	const op = "postgres.Incident.ListActive"

	const query = `
		SELECT id,
			   ST_Y(geo_point::geometry) AS lat,
			   ST_X(geo_point::geometry) AS lng,
			   radius_km,
			   status,
			   created_at
		FROM incidents
		WHERE status = 'active'
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		p.logger.Error("db query failed", slog.String("op", op), slog.Any("error", err))
		return nil, e.WrapError(ctx, op, err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		var inc domain.Incident
		if err := rows.Scan(
			&inc.ID,
			&inc.Lat,
			&inc.Lng,
			&inc.RadiusKM,
			&inc.Status,
			&inc.CreatedAt,
		); err != nil {
			p.logger.Error("row scan failed", slog.String("op", op), slog.Any("error", err))
			return nil, e.WrapError(ctx, op, err)
		}
		incidents = append(incidents, &inc)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error("rows err", slog.String("op", op), slog.Any("error", err))
		return nil, e.WrapError(ctx, op, err)
	}

	return incidents, nil
}
