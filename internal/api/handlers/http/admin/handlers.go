package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"redCollar/internal/domain"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

//go:generate mockgen -source=handlers.go -destination=mocks/mock.go
type AdminIncidents interface {
	Create(ctx context.Context, req domain.CreateIncidentRequest) (uuid.UUID, error)
	List(ctx context.Context, page, limit int) ([]*domain.Incident, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateIncidentRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type LocationChecker interface {
	CheckLocation(ctx context.Context, req domain.LocationCheckRequest) (domain.LocationCheckResponse, error)
}

type StatsGetter interface {
	GetStats(ctx context.Context, req domain.StatsRequest) (*domain.IncidentStats, error)
}

type Handler struct {
	logger          *slog.Logger
	Admin           AdminIncidents
	Stats           StatsGetter
	LocationChecker LocationChecker
}

func NewHandler(logger *slog.Logger, admin AdminIncidents, stats StatsGetter, locationChecker LocationChecker) *Handler {
	return &Handler{
		logger:          logger,
		Admin:           admin,
		Stats:           stats,
		LocationChecker: locationChecker,
	}
}

func (h *Handler) log(r *http.Request) *slog.Logger {
	reqID := chimw.GetReqID(r.Context())
	if reqID == "" {
		return h.logger
	}
	return h.logger.With(slog.String("request_id", reqID))
}

func (h *Handler) AdminIncidentCreate(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminIncidentCreate", slog.String("remote", r.RemoteAddr))

	var req domain.CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.Warn("invalid JSON", slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	l.Info("creating incident",
		slog.Float64("lat", req.Lat),
		slog.Float64("lng", req.Lng),
		slog.Float64("radius_km", req.RadiusKM),
		slog.String("status", string(req.Status)),
	)

	id, err := h.Admin.Create(r.Context(), req)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	l.Info("incident created", slog.String("id", id.String()))
	h.writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) AdminIncidentList(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminIncidentList", slog.String("query", r.URL.RawQuery), slog.String("remote", r.RemoteAddr))

	page := parseInt(r.URL.Query().Get("page"), 1)
	limit := parseInt(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
		l.Warn("limit capped", slog.Int("limit", limit))
	}

	incidents, total, err := h.Admin.List(r.Context(), page, limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	l.Info("incidents listed", slog.Int("count", len(incidents)), slog.Int64("total", total))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"incidents": incidents,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

func (h *Handler) AdminIncidentGet(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminIncidentGet", slog.String("remote", r.RemoteAddr))

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		l.Warn("invalid id", slog.String("id", idStr), slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	incident, err := h.Admin.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, incident)
}

func (h *Handler) AdminIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminIncidentUpdate", slog.String("remote", r.RemoteAddr))

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		l.Warn("invalid id", slog.String("id", idStr), slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req domain.UpdateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.Warn("invalid JSON", slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.Admin.Update(r.Context(), id, req); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdminIncidentDelete(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminIncidentDelete", slog.String("remote", r.RemoteAddr))

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		l.Warn("invalid id", slog.String("id", idStr), slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.Admin.Delete(r.Context(), id); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)
	l.Debug("AdminStats", slog.String("query", r.URL.RawQuery), slog.String("remote", r.RemoteAddr))

	minutesStr := r.URL.Query().Get("minutes") // query param
	if minutesStr == "" {
		minutesStr = "60"
	}

	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes <= 0 || minutes > 1440 {
		l.Warn("invalid minutes", slog.String("minutes", minutesStr))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "minutes must be 1-1440"})
		return
	}

	stats, err := h.Stats.GetStats(r.Context(), domain.StatsRequest{Minutes: minutes})
	if err != nil {
		l.Error("Stats.GetStats failed", slog.Any("error", err))
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	l.Info("stats success", slog.Int("minutes", minutes))
	h.writeJSON(w, http.StatusOK, stats)
}
