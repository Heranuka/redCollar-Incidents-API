package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"redCollar/internal/redis"
	"redCollar/pkg/e"

	"net/http"
	"time"

	"log/slog"
	"redCollar/internal/config"
	"redCollar/internal/domain"
)

type WebhookSender struct {
	logger *slog.Logger
	cfg    config.WebhookConfig
	queue  *redis.WebhookQueue
	http   *http.Client
}

func NewWebhookSender(logger *slog.Logger, cfg config.WebhookConfig, q *redis.WebhookQueue) *WebhookSender {
	return &WebhookSender{
		logger: logger,
		cfg:    cfg,
		queue:  q,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}
func (s *WebhookSender) Run(ctx context.Context) {
	s.logger.Info("webhookSender STARTED", slog.String("url", s.cfg.URL))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("webhookSender STOPPED", slog.String("reason", ctx.Err().Error()))
			return
		default:
		}

		payload, err := s.queue.BRPop(ctx, 5*time.Second)
		if err != nil {
			if errors.Is(err, e.ErrWebHookEmpty) {
				continue
			}
			s.logger.Error("BRPop failed", slog.Any("error", err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		s.logger.Info("sending webhook", slog.String("user_id", payload.UserID))
		s.sendWithRetry(ctx, payload)
	}
}
func (s *WebhookSender) sendWithRetry(ctx context.Context, p domain.WebhookPayload) {
	const maxRetries = 3

	body, err := json.Marshal(p)
	if err != nil {
		s.logger.Error("marshal webhook payload failed", slog.String("error", err.Error()))
		return
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			s.logger.Info("stop retries due to context cancel")
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.URL, bytes.NewReader(body))
		if err != nil {
			s.logger.Error("create webhook request failed", slog.String("error", err.Error()))
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := s.http.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		reason := "unknown"
		if err != nil {
			reason = err.Error()
		} else if resp != nil {
			reason = resp.Status
		}

		s.logger.Warn("webhook failed",
			slog.Int("attempt", attempt),
			slog.String("url", s.cfg.URL),
			slog.String("reason", reason),
		)

		time.Sleep(time.Duration(attempt) * time.Second)

	}
}
