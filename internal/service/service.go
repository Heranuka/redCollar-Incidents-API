package service

import (
	"context"
	"redCollar/internal/domain"

	"github.com/google/uuid"
)

//go:generate mockgen -source=service.go -destination=mocks/mock.go
type AdminIncidentService interface {
	Create(ctx context.Context, req domain.CreateIncidentRequest) (uuid.UUID, error)
	List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error) // ← ИСПРАВЛЕНО
	Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateIncidentRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}
type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) error
	List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Публичные use-case'ы
type PublicIncidentService interface {
	CheckLocation(ctx context.Context, req domain.LocationCheckRequest) (domain.LocationCheckResponse, error)
}

// Статистика
type StatsService interface {
	GetStats(ctx context.Context, req domain.StatsRequest) (*domain.IncidentStats, error)
}
type ctxKey string
type Service struct {
	AdminIncidentService  AdminIncidentService
	PublicIncidentService PublicIncidentService
	StatsService          StatsService
}

func NewService(
	adminIncidentService AdminIncidentService,
	publicIncidentService PublicIncidentService,
	statsService StatsService,
) *Service {
	return &Service{
		AdminIncidentService:  adminIncidentService,
		PublicIncidentService: publicIncidentService,
		StatsService:          statsService,
	}
}
