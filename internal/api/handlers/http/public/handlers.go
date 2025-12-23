package public

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"log/slog"
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
	var req domain.LocationCheckRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// ВАЖНО: запрещаем "лишние данные" после первого JSON-объекта
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	resp, err := h.PublicHandler.CheckLocation(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
