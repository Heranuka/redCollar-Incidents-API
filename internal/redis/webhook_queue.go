// internal/storage/redis/webhook_queue.go
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"redCollar/pkg/e"
	"time"

	"redCollar/internal/domain"

	"github.com/redis/go-redis/v9"
)

type WebhookQueue struct {
	client *redis.Client
	key    string
}

func NewWebhookQueue(client *redis.Client, key string) *WebhookQueue {
	return &WebhookQueue{client: client, key: key}
}

func (q *WebhookQueue) Enqueue(ctx context.Context, payload domain.WebhookPayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.key, b).Err()
}

func (q *WebhookQueue) BRPop(ctx context.Context, timeout time.Duration) (domain.WebhookPayload, error) {
	var p domain.WebhookPayload

	res, err := q.client.BRPop(ctx, timeout, q.key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return p, e.ErrWebHookEmpty
		}
		return p, err
	}
	if len(res) < 2 {
		return p, redis.Nil
	}
	if err := json.Unmarshal([]byte(res[1]), &p); err != nil {
		return p, err
	}
	return p, nil
}
