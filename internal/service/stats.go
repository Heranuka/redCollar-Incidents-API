package service

import (
	"context"
	"redCollar/internal/domain"
)

type StatsRepository interface {
	CountUniqueUsers(ctx context.Context, minutes int) (int64, error)
}

type statsService struct {
	repo StatsRepository
}

func NewStatsService(repo StatsRepository) StatsService {
	return &statsService{repo: repo}
}

func (s *statsService) GetStats(ctx context.Context, req domain.StatsRequest) (*domain.IncidentStats, error) {
	minutes := req.Minutes
	if minutes == 0 {
		minutes = 60
	}

	cnt, err := s.repo.CountUniqueUsers(ctx, minutes)
	if err != nil {
		return nil, err
	}

	return &domain.IncidentStats{UserCount: cnt}, nil
}
