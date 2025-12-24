package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"redCollar/pkg/e"
	"strconv"
)

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	l := h.log(r)

	l.Error("handler error",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	)

	switch {
	case errors.Is(err, e.ErrNotFound):
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, e.ErrInvalidInput):
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
	case errors.Is(err, e.ErrConflict):
		h.writeJSON(w, http.StatusConflict, map[string]string{"error": "conflict"})
	default:
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
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
