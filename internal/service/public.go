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
	s.logger.Info("location check START",
		slog.String("user_id", req.UserID),
		slog.Float64("lat", req.Lat),
		slog.Float64("lng", req.Lng),
	)

	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		s.logger.Warn("invalid coordinates",
			slog.String("user_id", req.UserID),
			slog.Float64("lat", req.Lat),
			slog.Float64("lng", req.Lng),
		)
		return domain.LocationCheckResponse{}, e.ErrInvalidCoordinates
	}

	incidents, err := s.cache.GetActive(ctx)
	if err != nil {
		s.logger.Error("cache.GetActive failed", slog.Any("error", err))
		return domain.LocationCheckResponse{}, err
	}
	s.logger.Debug("cache loaded", slog.Int("active_incidents", len(incidents)))

	// ✅ фикс: проверяем по радиусу каждого инцидента
	nearby := filterNearby(incidents, req.Lat, req.Lng)
	s.logger.Info("haversine filter done",
		slog.Int("total", len(incidents)),
		slog.Int("nearby", len(nearby)),
	)

	ids := make([]uuid.UUID, 0, len(nearby))
	for _, inc := range nearby {
		ids = append(ids, inc.ID)
	}

	if len(ids) > 0 {
		payload := domain.WebhookPayload{
			UserID:    req.UserID,
			Lat:       req.Lat,
			Lng:       req.Lng,
			Incidents: ids,
			CheckedAt: time.Now().UTC(),
		}
		if err := s.webhookQueue.Enqueue(ctx, payload); err != nil {
			s.logger.Error("enqueue webhook failed", slog.Any("error", err))
		} else {
			s.logger.Info("webhook enqueued", slog.String("user_id", req.UserID), slog.Int("incidents", len(ids)))
		}
	} else {
		s.logger.Debug("no incidents nearby")
	}

	s.logger.Info("location check END", slog.Int("incidents_found", len(ids)))
	return domain.LocationCheckResponse{Incidents: idsToStrings(ids)}, nil
}

func idsToStrings(ids []uuid.UUID) []string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}
func filterNearby(incidents []domain.CachedIncident, lat, lng float64) []domain.NearbyIncident {
	nearby := make([]domain.NearbyIncident, 0)
	for _, inc := range incidents {
		dist := haversine(lat, lng, inc.Lat, inc.Lng)

		// ✅ радиус берём из инцидента
		if dist <= inc.RadiusKM {
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
