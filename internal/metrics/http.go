package metrics

import (
	"encoding/json"
	"net/http"

	"github.com/ramdns/ramdns/internal/upstream"
)

type HealthProvider interface {
	Health() map[string]upstream.ServerHealth
}

type Handler struct {
	metrics  *Metrics
	upstream HealthProvider
}

type Response struct {
	CacheHits     uint64  `json:"cache_hits"`
	CacheMisses   uint64  `json:"cache_misses"`
	UpstreamQuery uint64  `json:"upstream_queries"`
	UpstreamError uint64  `json:"upstream_errors"`
	TotalQueries  uint64  `json:"total_queries"`
	CacheHitRatio float64 `json:"cache_hit_ratio"`
}

func NewHandler(m *Metrics, providers ...HealthProvider) *Handler {
	h := &Handler{
		metrics: m,
	}

	if len(providers) > 0 {
		h.upstream = providers[0]
	}

	return h
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/metrics" {
		http.NotFound(w, r)
		return
	}

	s := h.metrics.Snapshot()

	total := s.CacheHits + s.CacheMisses

	var ratio float64
	if total > 0 {
		ratio = float64(s.CacheHits) / float64(total)
	}

	response := Response{
		CacheHits:     s.CacheHits,
		CacheMisses:   s.CacheMisses,
		UpstreamQuery: s.UpstreamQuery,
		UpstreamError: s.UpstreamError,
		TotalQueries:  total,
		CacheHitRatio: ratio,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}
