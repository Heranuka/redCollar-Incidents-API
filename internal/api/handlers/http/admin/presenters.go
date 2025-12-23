package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"redCollar/pkg/e"
	"strconv"
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
