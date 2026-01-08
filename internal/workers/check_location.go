package workers

import (
	"context"
	"log"
	"redCollar/internal/domain"
	"sync"
	"time"
)

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
	incidents IncidentCacheService
	jobs      chan CheckLocationJob
	poolSize  int
}

func NewLocationChecker(incidents IncidentCacheService, poolSize int) *LocationChecker {
	return &LocationChecker{
		incidents: incidents,
		jobs:      make(chan CheckLocationJob, 100),
		poolSize:  poolSize,
	}
}

func (w *LocationChecker) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < w.poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.worker(ctx, w.jobs)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.producer(ctx)
	}()
	wg.Wait()
}

func (w *LocationChecker) producer(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := w.incidents.GetActive(ctx)
			if err != nil {
				log.Print("запрос сдох у GetActive в check_location.go/producer")
				continue
			}
		}
	}
}

func (w *LocationChecker) worker(ctx context.Context, jobs <-chan CheckLocationJob) {
	for {
		select {
		case <-ctx.Done():
			return
		default: // Чтобы не блокироваться
			select {
			case job := <-jobs:
				w.processJob(ctx, job)
			case <-ctx.Done():
				return
			}
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
	var result []domain.NearbyIncident
	return result
}
