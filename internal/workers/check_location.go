package workers

import (
	"context"
	"redCollar/internal/domain"
	"time"
)

// internal/workers/check_location.go
type IncidentCacheService interface {
	GetActive(ctx context.Context) ([]domain.CachedIncident, error)
	SetActive(ctx context.Context, incidents []domain.CachedIncident, ttl time.Duration) error
}
type CheckLocationJob struct {
	Lat        float64
	Lng        float64
	ResultChan chan<- []domain.NearbyIncident
	Timeout    time.Duration
}

type LocationChecker struct {
	incidents IncidentCacheService // из service
	jobs      chan CheckLocationJob
	cancel    context.CancelFunc
	poolSize  int
}

func NewLocationChecker(incidents IncidentCacheService, poolSize int) *LocationChecker {
	return &LocationChecker{
		incidents: incidents,
		jobs:      make(chan CheckLocationJob, 100),
		poolSize:  poolSize,
	}
}

func (w *LocationChecker) Start(ctx context.Context) {
	// 1. Запускаем воркеров
	for i := 0; i < w.poolSize; i++ {
		go w.worker(ctx, w.jobs)
	}

	// 2. Producer: периодически обновляет incidents из Redis
	go w.producer(ctx)
}

func (w *LocationChecker) producer(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Обновляем кэш активных инцидентов
			_, err := w.incidents.GetActive(ctx)
			if err != nil {
				return
			}
		}
	}
}

func (w *LocationChecker) worker(ctx context.Context, jobs <-chan CheckLocationJob) {
	for {
		select {
		case job := <-jobs:
			w.processJob(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (w *LocationChecker) processJob(ctx context.Context, job CheckLocationJob) {
	incidents, err := w.incidents.GetActive(ctx)
	if err != nil {
		job.ResultChan <- nil
		close(job.ResultChan)
		return
	}

	nearby := filterNearby(incidents, job.Lat, job.Lng)

	select {
	case job.ResultChan <- nearby:
	case <-time.After(job.Timeout):
	case <-ctx.Done():
	}
	close(job.ResultChan)
}

func filterNearby(incidents []domain.CachedIncident, lat, lng float64) []domain.NearbyIncident {
	// твоя логика расстояния
	var result []domain.NearbyIncident
	return result
}

func (w *LocationChecker) Stop() {
	w.cancel() // ✅ Graceful shutdown
}
