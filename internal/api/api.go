package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ramdns/ramdns/internal/metrics"
)

type Handler struct {
	startedAt time.Time
	metrics   *metrics.Metrics
}

func NewHandler(
	startedAt time.Time,
	m *metrics.Metrics,
) *Handler {
	return &Handler{
		startedAt: startedAt,
		metrics:   m,
	}
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.URL.Path {
	case "/api/v1/health":
		h.health(w)
	case "/api/v1/metrics":
		h.metricsHandler(w)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) health(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"service":    "ramdns",
		"started_at": h.startedAt.UTC().Format(time.RFC3339),
		"uptime":     time.Since(h.startedAt).Round(time.Second).String(),
	})
}

func (h *Handler) metricsHandler(w http.ResponseWriter) {
	if h.metrics == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "metrics unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
