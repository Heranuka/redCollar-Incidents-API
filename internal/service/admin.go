package service

import (
	"context"

	"redCollar/internal/domain"

	"github.com/google/uuid"
)

type AdminService struct {
	repo IncidentRepository
}

func NewAdminIncidentService(repo IncidentRepository) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) Create(ctx context.Context, req domain.CreateIncidentRequest) (uuid.UUID, error) {
	inc := &domain.Incident{
		ID:       uuid.New(),
		Lat:      req.Lat,
		Lng:      req.Lng,
		RadiusKM: req.RadiusKM,
		Status:   domain.IncidentActive,
	}
	if err := s.repo.Create(ctx, inc); err != nil {
		return uuid.Nil, err
	}
	return inc.ID, nil
}
func (s *AdminService) List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error) { // ← ИСПРАВЬ сигнатуру
	items, total, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil // ← возвращаем как есть, без toIncidents
}

func (s *AdminService) Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	return s.repo.Get(ctx, id)
}

func (s *AdminService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateIncidentRequest) error {
	inc, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if req.Lat != nil {
		inc.Lat = *req.Lat
	}
	if req.Lng != nil {
		inc.Lng = *req.Lng
	}
	if req.RadiusKM != nil {
		inc.RadiusKM = *req.RadiusKM
	}
	if req.Status != nil {
		inc.Status = *req.Status
	}
	return s.repo.Update(ctx, inc)
}

func (s *AdminService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func toIncidents(src []*domain.Incident) []domain.Incident {
	out := make([]domain.Incident, 0, len(src))
	for _, p := range src {
		out = append(out, *p)
	}
	return out
}
