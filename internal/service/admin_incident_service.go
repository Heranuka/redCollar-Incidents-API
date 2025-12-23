package service

import (
	"context"

	"redCollar/internal/domain"

	"github.com/google/uuid"
)

func (s *Service) Create(ctx context.Context, req domain.CreateIncidentRequest) (uuid.UUID, error) {
	return s.AdminIncidentService.Create(ctx, req)
}

func (s *Service) List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error) {
	return s.AdminIncidentService.List(ctx, page, limit)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	return s.AdminIncidentService.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req domain.UpdateIncidentRequest) error {
	return s.AdminIncidentService.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.AdminIncidentService.Delete(ctx, id)
}
