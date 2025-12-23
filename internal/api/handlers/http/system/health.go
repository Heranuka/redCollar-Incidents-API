package system

import (
	"net/http"

	"log/slog"
)

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) SystemHealth(w http.ResponseWriter, r *http.Request) {
	// Минимальный health endpoint: важен статус-код 200. [web:44]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // status задаётся через WriteHeader [web:48]
	_, _ = w.Write([]byte("ok"))
}
