package management

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ramdns/ramdns/internal/metrics"
	"github.com/ramdns/ramdns/internal/upstream"
)

type Handler struct {
	metrics  *metrics.Metrics
	upstream *upstream.Resolver
	token    string
}

type StatusResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	API     string `json:"api"`
}

func NewHandler(m *metrics.Metrics, u *upstream.Resolver, token string) *Handler {
	return &Handler{
		metrics:  m,
		upstream: u,
		token:    token,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.URL.Path {
	case "/api/v1/status":
		h.status(w)
	case "/api/v1/metrics":
		h.writeJSON(w, h.metrics.Snapshot())
	case "/api/v1/upstreams":
		h.writeJSON(w, h.upstream.Health())
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) authorized(r *http.Request) bool {
	if h.token == "" {
		return false
	}

	const prefix = "Bearer "
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, prefix) {
		return false
	}

	provided := strings.TrimSpace(strings.TrimPrefix(value, prefix))
	if len(provided) != len(h.token) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) == 1
}

func (h *Handler) status(w http.ResponseWriter) {
	h.writeJSON(w, StatusResponse{
		Status:  "ok",
		Service: "ramdns",
		API:     "v1",
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}
