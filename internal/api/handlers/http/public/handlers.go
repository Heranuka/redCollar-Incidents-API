package public

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"redCollar/internal/domain"
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
