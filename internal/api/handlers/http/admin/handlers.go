package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"redCollar/internal/domain"

	"github.com/go-chi/chi/v5"
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

func NewHandler(logger *slog.Logger, Admin AdminIncidents, Stats StatsGetter, LocationChecker LocationChecker) *Handler {
	return &Handler{
		logger:          logger,
		Admin:           Admin,
		Stats:           Stats,
		LocationChecker: LocationChecker,
	}
}

func (h *Handler) AdminIncidentCreate(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	id, err := h.Admin.Create(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) AdminIncidentList(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"), 1)
	limit := parseInt(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	incidents, total, err := h.Admin.List(r.Context(), page, limit) // ← вот так
	if err != nil {
		h.handleError(w, err)
		return
	}

	resp := map[string]any{
		"incidents": incidents,
		"total":     total,
		"page":      page,
		"limit":     limit,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminIncidentGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id") // работает для роутов вида /incidents/{id} [web:3]
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	incident, err := h.Admin.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, incident)
}

func (h *Handler) AdminIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id") // [web:3]
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req domain.UpdateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.Admin.Update(r.Context(), id, req); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdminIncidentDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id") // [web:3]
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.Admin.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	minutes := parseInt(r.URL.Query().Get("minutes"), 60)

	req := domain.StatsRequest{Minutes: minutes}

	stats, err := h.Stats.GetStats(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}
