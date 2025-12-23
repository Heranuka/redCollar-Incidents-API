package service

import (
	"context"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"redCollar/internal/domain"
	"redCollar/pkg/e"
)

type IncidentGeoRepository interface {
	FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]uuid.UUID, error)
	SaveCheck(ctx context.Context, check *domain.LocationCheck) error
}

type WebhookQueue interface {
	Enqueue(ctx context.Context, payload domain.WebhookPayload) error
}

type publicIncidentService struct {
	repo            IncidentGeoRepository
	webhookQueue    WebhookQueue
	logger          *slog.Logger
	defaultRadiusKm float64
}

func NewPublicIncidentService(
	repo IncidentGeoRepository,
	q WebhookQueue,
	logger *slog.Logger,
	defaultRadiusKm float64,
) PublicIncidentService {
	if defaultRadiusKm <= 0 {
		defaultRadiusKm = 1.0
	}
	return &publicIncidentService{
		repo:            repo,
		webhookQueue:    q,
		logger:          logger,
		defaultRadiusKm: defaultRadiusKm,
	}
}

func (s *publicIncidentService) CheckLocation(
	ctx context.Context,
	req domain.LocationCheckRequest,
) (domain.LocationCheckResponse, error) {
	// базовая бизнес‑валидация
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		return domain.LocationCheckResponse{}, e.ErrInvalidCoordinates
	}

	// 1. ищем ближайшие инциденты
	ids, err := s.repo.FindNearby(ctx, req.Lat, req.Lng, s.defaultRadiusKm)
	if err != nil {
		s.logger.Error("FindNearby failed", slog.Any("err", err))
		return domain.LocationCheckResponse{}, err
	}
	incidents := make([]string, len(ids))
	for i, id := range ids {
		incidents[i] = id.String()
	}

	// 2. сохраняем факт проверки
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return domain.LocationCheckResponse{}, e.ErrInvalidUserID
	}

	check := &domain.LocationCheck{
		UserID:      userID,
		Lat:         req.Lat,
		Lng:         req.Lng,
		IncidentIDs: ids,
		CheckedAt:   time.Now().UTC(),
	}

	if err := s.repo.SaveCheck(ctx, check); err != nil {
		s.logger.Error("SaveCheck failed", slog.Any("err", err))
		// не роняем ответ
	}

	// 3. если есть инциденты — кладём задачу в очередь вебхуков
	if len(ids) > 0 {
		payload := domain.WebhookPayload{
			UserID:    req.UserID,
			Lat:       req.Lat,
			Lng:       req.Lng,
			Incidents: ids,
			CheckedAt: check.CheckedAt,
		}
		if err := s.webhookQueue.Enqueue(ctx, payload); err != nil {
			s.logger.Error("Enqueue webhook failed", slog.Any("err", err))
		}
	}

	// 4. возвращаем ответ клиенту
	return domain.LocationCheckResponse{Incidents: incidents}, nil
}
