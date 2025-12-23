package service

import (
	"context"
	"redCollar/internal/domain"
)

func (s *Service) CheckLocation(ctx context.Context, req domain.LocationCheckRequest) (domain.LocationCheckResponse, error) {
	return s.PublicIncidentService.CheckLocation(ctx, req)
}
