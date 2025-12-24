package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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
func InitRouter(cfg *config.Config, adminHandler *admin.Handler, publicHandler *public.Handler, systemHandler *system.Handler, logger *slog.Logger) *chi.Mux {
	r := chi.NewMux()

	// чтобы request_id попал в лог chi.Logger
	r.Use(chimw.RequestID) // [web:243]
	r.Use(chimw.Recoverer)
	r.Use(chimw.Logger) // [web:243]

	r.Route("/api/v1", func(api chi.Router) {
		// ADMIN
		api.Route("/admin", func(ar chi.Router) {
			ar.Use(middleware.APIKeyMiddleware(cfg.APIKey))
			ar.Use(middleware.Limit(2, 5, 10*time.Minute, logger))

			// ✅ stats НЕ внутри incidents
			ar.Get("/stats", adminHandler.AdminStats)

			ar.Route("/incidents", func(ir chi.Router) {
				ir.Post("/", adminHandler.AdminIncidentCreate)
				ir.Get("/", adminHandler.AdminIncidentList)

				ir.Route("/{id}", func(rr chi.Router) {
					rr.Get("/", adminHandler.AdminIncidentGet)
					rr.Put("/", adminHandler.AdminIncidentUpdate)
					rr.Delete("/", adminHandler.AdminIncidentDelete)
				})
			})
		})

		// PUBLIC
		api.Route("/location", func(pr chi.Router) {
			pr.Use(middleware.Limit(10, 20, 5*time.Minute, logger))
			pr.Post("/check", publicHandler.PublicLocationCheck)
		})

		// SYSTEM
		api.Get("/health", systemHandler.SystemHealth)
	})

	return r
}
func (s *Server) Run(ctx context.Context) error {
	port := s.cfg.Http.Port
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:         port,
		Handler:      s.router,
		ReadTimeout:  s.cfg.Http.ReadTimeout,
		WriteTimeout: s.cfg.Http.WriteTimeout,
		IdleTimeout:  30 * time.Second,
	}

	errChan := make(chan error, 1)

	go func() {
		s.logger.Info("🚀 Starting HTTP server",
			slog.String("addr", srv.Addr),
			slog.Duration("read_timeout", s.cfg.Http.ReadTimeout),
			slog.Duration("write_timeout", s.cfg.Http.WriteTimeout),
		)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("ListenAndServe error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("🛑 Shutting down HTTP server", slog.String("reason", ctx.Err().Error()))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.Http.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown failed", slog.Any("error", err))
			return err
		}
		return nil

	case err := <-errChan:
		return err
	}
}
