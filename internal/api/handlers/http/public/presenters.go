package public

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"redCollar/pkg/e"
	"strconv"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// В Handler добавь методы:
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var status int
	switch {
	case errors.Is(err, e.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, e.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, e.ErrConflict):
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}
	h.writeJSON(w, status, map[string]string{"error": err.Error()})
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
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
