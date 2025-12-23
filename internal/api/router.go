package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"redCollar/internal/api/handlers/http/admin"
	"redCollar/internal/api/handlers/http/public"
	"redCollar/internal/api/handlers/http/system"
	"redCollar/internal/config"
	"redCollar/internal/middleware"
	"redCollar/internal/service"
)

type Server struct {
	logger *slog.Logger
	router *chi.Mux
	cfg    config.Config
}

func NewServer(cfg *config.Config, logger *slog.Logger, svc *service.Service) *Server {
	adminHandler := admin.NewHandler(logger, svc.AdminIncidentService, svc.StatsService, svc.PublicIncidentService)
	publicHandler := public.NewHandler(logger, svc.PublicIncidentService)
	systemHandler := system.NewHandler(logger)

	r := InitRouter(cfg, adminHandler, publicHandler, systemHandler, logger)

	return &Server{
		logger: logger,
		router: r,
		cfg:    *cfg,
	}
}

func InitRouter(
	cfg *config.Config,
	adminHandler *admin.Handler,
	publicHandler *public.Handler,
	systemHandler *system.Handler,
	logger *slog.Logger,
) *chi.Mux {
	r := chi.NewMux()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	r.Route("/api/v1", func(api chi.Router) {
		// ADMIN
		api.Route("/admin", func(ar chi.Router) {
			ar.Use(middleware.APIKeyMiddleware(cfg.APIKey))
			ar.Use(middleware.Limit(2, 5, 10*time.Minute, logger))

			ar.Route("/incidents", func(ir chi.Router) {
				ir.Post("/", adminHandler.AdminIncidentCreate)
				ir.Get("/", adminHandler.AdminIncidentList)

				ir.Route("/{id}", func(rr chi.Router) {
					rr.Get("/", adminHandler.AdminIncidentGet)
					rr.Put("/", adminHandler.AdminIncidentUpdate)
					rr.Delete("/", adminHandler.AdminIncidentDelete)
				})
			})

			ar.Get("/incidents/stats", adminHandler.AdminStats)
		})

		// PUBLIC
		api.Route("/location", func(pr chi.Router) {
			pr.Use(middleware.Limit(10, 20, 5*time.Minute, logger))
			pr.Post("/check", publicHandler.PublicLocationCheck)
		})

		// SYSTEM
		api.Get("/system/health", systemHandler.SystemHealth)
	})

	return r
}

func (s *Server) Run(ctx context.Context) error {
	errChan := make(chan error, 1)

	srv := &http.Server{
		Addr:    ":" + s.cfg.Http.Port,
		Handler: s.router,
	}

	go func() {
		s.logger.Info("Starting listening", slog.String("port", s.cfg.Http.Port))

		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("ListenAndServe error: %w", err)
			return
		}
		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down the server", slog.String("reason", ctx.Err().Error()))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)

	case err := <-errChan:
		return err
	}
}
