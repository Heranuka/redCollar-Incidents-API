package service

import (
	"context"
	"log/slog"
	"time"

	"redCollar/internal/domain"

	"github.com/google/uuid"
)

type AdminService struct {
	repo  IncidentRepository
	cache IncidentCacheService
}

func NewAdminIncidentService(repo IncidentRepository, cache IncidentCacheService) *AdminService {
	return &AdminService{repo: repo, cache: cache}
}

func (s *AdminService) Create(ctx context.Context, req domain.CreateIncidentRequest) (uuid.UUID, error) {
	status := req.Status
	if status == "" {
		status = domain.IncidentActive
	}
	inc := &domain.Incident{
		ID:       uuid.New(),
		Lat:      req.Lat,
		Lng:      req.Lng,
		RadiusKM: req.RadiusKM,
		Status:   status,
	}
	if err := s.repo.Create(ctx, inc); err != nil {
		return uuid.Nil, err
	}
	s.refreshCache(ctx)
	return inc.ID, nil
}
func (s *AdminService) List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error) {
	items, total, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
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
	if err := s.repo.Update(ctx, inc); err != nil {
		return err
	}
	s.refreshCache(ctx)
	return nil
}

func (s *AdminService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.refreshCache(ctx)
	return nil
}

func toIncidents(src []*domain.Incident) []domain.Incident {
	out := make([]domain.Incident, 0, len(src))
	for _, p := range src {
		out = append(out, *p)
	}
	return out
}
func (s *AdminService) refreshCache(ctx context.Context) {
	incidents, err := s.repo.ListActive(ctx)
	if err != nil {
		slog.Default().Error("refreshCache: repo.ListActive failed", slog.Any("error", err))
		return
	}

	cached := make([]domain.CachedIncident, 0, len(incidents))
	for _, inc := range incidents {
		cached = append(cached, domain.CachedIncident{
			ID:       inc.ID,
			Lat:      inc.Lat,
			Lng:      inc.Lng,
			RadiusKM: inc.RadiusKM,
		})
	}

	if err := s.cache.SetActive(ctx, cached, 5*time.Minute); err != nil {
		slog.Default().Error("refreshCache: cache.SetActive failed", slog.Any("error", err))
		return
	}
}
