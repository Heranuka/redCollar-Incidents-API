package postgres

import (
	"context"
	"redCollar/internal/domain"

	"github.com/google/uuid"
)

type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) error
	List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error // soft delete
	ListActive(ctx context.Context) ([]*domain.Incident, error)
}

type StatsRepository interface {
	SaveCheck(ctx context.Context, check *domain.LocationCheck) error
	CountUniqueUsers(ctx context.Context, minutes int) (int64, error)
}

type GeoRepository interface {
	FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]uuid.UUID, error)
	SaveCheck(ctx context.Context, check *domain.LocationCheck) error
}

func (p *Postgres) AdminIncidents() IncidentRepository { return p.IncidentAdmin }
func (p *Postgres) PublicIncidents() GeoRepository     { return p.Geo }
func (p *Postgres) Stats() StatsRepository             { return p.Stat }
