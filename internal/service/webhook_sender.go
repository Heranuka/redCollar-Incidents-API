package service

import (
	"bytes"
	"context"
	"encoding/json"

	"net/http"
	"time"

	"log/slog"
	"redCollar/internal/config"
	"redCollar/internal/domain"
	"redCollar/internal/storage/redis"
)

type WebhookSender struct {
	logger *slog.Logger
	cfg    config.WebhookConfig
	queue  *redis.WebhookQueue // ← ИЗМЕНЕНО: Queue → WebhookQueue
	http   *http.Client
}

func NewWebhookSender(logger *slog.Logger, cfg config.WebhookConfig, q *redis.WebhookQueue) *WebhookSender { // ← ИЗМЕНЕНО
	return &WebhookSender{
		logger: logger,
		cfg:    cfg,
		queue:  q,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *WebhookSender) Run(ctx context.Context) {
	for {
		payload, err := s.queue.BRPop(ctx, 5*time.Second)
		if err != nil {
			// WebhookQueue уже возвращает redis.Nil при пустой очереди
			s.logger.Debug("queue empty, waiting...")
			continue
		}

		s.sendWithRetry(ctx, payload)
	}
}

func (s *WebhookSender) sendWithRetry(ctx context.Context, p domain.WebhookPayload) {
	const maxRetries = 3

	body, _ := json.Marshal(p)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.URL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.http.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		s.logger.Warn("webhook failed",
			slog.Int("attempt", attempt),
			slog.String("url", s.cfg.URL),
		)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
}
