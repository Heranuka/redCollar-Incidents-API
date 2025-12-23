package service

import (
	"context"
	"redCollar/internal/domain"
)

func (s *Service) GetStats(ctx context.Context, req domain.StatsRequest) (*domain.IncidentStats, error) {
	return s.StatsService.GetStats(ctx, req)
}
