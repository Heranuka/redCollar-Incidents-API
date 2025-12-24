package public

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"redCollar/internal/domain"

	chimw "github.com/go-chi/chi/v5/middleware"
)

//go:generate mockgen -source=handlers.go -destination=mocks/mock.go
type PublicHandler interface {
	CheckLocation(ctx context.Context, req domain.LocationCheckRequest) (domain.LocationCheckResponse, error)
}

type Handler struct {
	logger        *slog.Logger
	PublicHandler PublicHandler
}

func NewHandler(logger *slog.Logger, publicHandler PublicHandler) *Handler {
	return &Handler{
		logger:        logger,
		PublicHandler: publicHandler,
	}
}

func (h *Handler) log(r *http.Request) *slog.Logger {
	reqID := chimw.GetReqID(r.Context())
	if reqID == "" {
		return h.logger
	}
	return h.logger.With(slog.String("request_id", reqID))
}

func (h *Handler) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("json encode failed", slog.Any("error", err))
	}
}

func (h *Handler) PublicLocationCheck(w http.ResponseWriter, r *http.Request) {
	l := h.log(r)

	l.Debug("PublicLocationCheck called",
		slog.String("remote", r.RemoteAddr),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)

	var req domain.LocationCheckRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		l.Warn("invalid JSON", slog.Any("error", err))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// запрещаем мусор после JSON
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		l.Warn("extra data after JSON", slog.Any("error", err))
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	l.Info("checking location",
		slog.Float64("lat", req.Lat),
		slog.Float64("lng", req.Lng),
	)

	resp, err := h.PublicHandler.CheckLocation(r.Context(), req)
	if err != nil {
		l.Error("check location failed", slog.Any("error", err))
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	l.Info("check location success")
	h.writeJSON(w, http.StatusOK, resp)
}
