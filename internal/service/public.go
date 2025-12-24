package service

import (
	"context"
	"math"
	"time"

	"log/slog"

	"redCollar/internal/domain"
	"redCollar/pkg/e"

	"github.com/google/uuid"
)

type IncidentGeoRepository interface {
	FindNearby(ctx context.Context, lat, lng, radiusKm float64) ([]uuid.UUID, error)
	SaveCheck(ctx context.Context, check *domain.LocationCheck) error
}

type WebhookQueue interface {
	Enqueue(ctx context.Context, payload domain.WebhookPayload) error
}

type publicIncidentService struct {
	cache           IncidentCacheService
	webhookQueue    WebhookQueue
	logger          *slog.Logger
	defaultRadiusKm float64
}

func NewPublicIncidentService(
	cache IncidentCacheService,
	q WebhookQueue,
	logger *slog.Logger,
	defaultRadiusKm float64,
) PublicIncidentService {
	if defaultRadiusKm <= 0 {
		defaultRadiusKm = 1.0
	}
	return &publicIncidentService{
		cache:           cache,
		webhookQueue:    q,
		logger:          logger,
		defaultRadiusKm: defaultRadiusKm,
	}
}

func (s *publicIncidentService) CheckLocation(ctx context.Context, req domain.LocationCheckRequest) (domain.LocationCheckResponse, error) {
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		return domain.LocationCheckResponse{}, e.ErrInvalidCoordinates
	}

	// 1. ✅ Redis: АКТИВНЫЕ инциденты (0.1ms)
	incidents, err := s.cache.GetActive(ctx)
	if err != nil {
		s.logger.Error("cache.GetActive failed", slog.Any("err", err))
		return domain.LocationCheckResponse{}, err
	}

	// 2. ✅ CPU: фильтр по расстоянию (0.1ms)
	nearby := filterNearby(incidents, req.Lat, req.Lng, s.defaultRadiusKm)

	// 3. ids для ответа
	ids := make([]uuid.UUID, 0, len(nearby))
	for _, inc := range nearby {
		ids = append(ids, inc.ID)
	}

	incidentsStr := make([]string, len(ids))
	for i, id := range ids {
		incidentsStr[i] = id.String()
	}

	// 4. SaveCheck + Webhook (как было)
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

	return domain.LocationCheckResponse{Incidents: incidentsStr}, nil
}

// В service/utils.go
func filterNearby(incidents []domain.CachedIncident, lat, lng, radiusKm float64) []domain.NearbyIncident {
	var nearby []domain.NearbyIncident
	for _, inc := range incidents {
		dist := haversine(lat, lng, inc.Lat, inc.Lng)
		if dist <= radiusKm {
			nearby = append(nearby, domain.NearbyIncident{
				ID:         inc.ID,
				Lat:        inc.Lat,
				Lng:        inc.Lng,
				RadiusKM:   inc.RadiusKM,
				DistanceKM: dist,
			})
		}
	}
	return nearby
}
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Радиус Земли в км

	dLat := deg2rad(lat2 - lat1)
	dLon := deg2rad(lon2 - lon1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func deg2rad(deg float64) float64 {
	return deg * math.Pi / 180.0
}
