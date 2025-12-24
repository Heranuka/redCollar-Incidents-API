package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"redCollar/internal/domain"

	goredis "github.com/redis/go-redis/v9"
)

type IncidentCacheService interface {
	GetActive(ctx context.Context) ([]domain.CachedIncident, error)
	SetActive(ctx context.Context, incidents []domain.CachedIncident, ttl time.Duration) error
}

type IncidentCache struct {
	client *goredis.Client
	key    string
}

func NewIncidentCache(r *Redis) *IncidentCache {
	return &IncidentCache{
		client: r.Client,
		key:    "incidents:active",
	}
}

func (c *IncidentCache) GetActive(ctx context.Context) ([]domain.CachedIncident, error) {
	data, err := c.client.Get(ctx, c.key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var incidents []domain.CachedIncident
	if err := json.Unmarshal(data, &incidents); err != nil {
		return nil, err
	}

	return incidents, nil
}

func (c *IncidentCache) SetActive(ctx context.Context, incidents []domain.CachedIncident, ttl time.Duration) error {
	b, err := json.Marshal(incidents)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key, b, ttl).Err()
}
