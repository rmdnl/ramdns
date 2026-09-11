package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ramdns/ramdns/internal/metrics"
	"github.com/ramdns/ramdns/internal/store"
)

type Handler struct {
	startedAt time.Time
	metrics   *metrics.Metrics
	store     *store.Store
}

func NewHandler(
	startedAt time.Time,
	m *metrics.Metrics,
	s *store.Store,
) *Handler {
	return &Handler{
		startedAt: startedAt,
		metrics:   m,
		store:     s,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/health":
		h.health(w)

	case r.URL.Path == "/api/v1/metrics":
		h.metricsHandler(w)

	case r.URL.Path == "/api/v1/settings" && r.Method == http.MethodGet:
		h.listSettings(w, r)

	case strings.HasPrefix(r.URL.Path, "/api/v1/settings/") && r.Method == http.MethodPut:
		h.setSetting(w, r)

	case r.URL.Path == "/api/v1/adlists" && r.Method == http.MethodGet:
		h.listAdlists(w, r)

	case r.URL.Path == "/api/v1/adlists" && r.Method == http.MethodPost:
		h.createAdlist(w, r)

	case strings.HasPrefix(r.URL.Path, "/api/v1/adlists/") && r.Method == http.MethodPatch:
		h.setAdlistEnabled(w, r)

	case strings.HasPrefix(r.URL.Path, "/api/v1/adlists/") && r.Method == http.MethodDelete:
		h.deleteAdlist(w, r)

	case r.URL.Path == "/api/v1/rules" && r.Method == http.MethodGet:
		h.listRules(w, r)

	case r.URL.Path == "/api/v1/rules" && r.Method == http.MethodPost:
		h.createRule(w, r)

	case strings.HasPrefix(r.URL.Path, "/api/v1/rules/") && r.Method == http.MethodPatch:
		h.setRuleEnabled(w, r)

	case strings.HasPrefix(r.URL.Path, "/api/v1/rules/") && r.Method == http.MethodDelete:
		h.deleteRule(w, r)

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

func (h *Handler) listSettings(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	rows, err := h.store.ListSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) setSetting(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/api/v1/settings/")
	if key == "" || strings.Contains(key, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid setting key",
		})
		return
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.SetSetting(r.Context(), key, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	setting, err := h.store.GetSetting(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, setting)
}

func (h *Handler) listAdlists(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	rows, err := h.store.ListAdlists(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) createAdlist(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	var req struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled *bool  `json:"enabled"`
	}

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	adlist, err := h.store.CreateAdlist(
		r.Context(),
		req.Name,
		req.URL,
		enabled,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, adlist)
}

func (h *Handler) setAdlistEnabled(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	id, err := pathID(r.URL.Path, "/api/v1/adlists/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.SetAdlistEnabled(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	adlist, err := h.store.GetAdlist(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, adlist)
}

func (h *Handler) deleteAdlist(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	id, err := pathID(r.URL.Path, "/api/v1/adlists/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.DeleteAdlist(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	rows, err := h.store.ListCustomRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) createRule(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	var req struct {
		Domain   string `json:"domain"`
		RuleType string `json:"rule_type"`
		Enabled  *bool  `json:"enabled"`
	}

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule, err := h.store.CreateCustomRule(
		r.Context(),
		req.Domain,
		req.RuleType,
		enabled,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) setRuleEnabled(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	id, err := pathID(r.URL.Path, "/api/v1/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.SetCustomRuleEnabled(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	rule, err := h.store.GetCustomRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) deleteRule(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "store unavailable",
		})
		return
	}

	id, err := pathID(r.URL.Path, "/api/v1/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.store.DeleteCustomRule(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(path, prefix string) (int64, error) {
	value := strings.TrimPrefix(path, prefix)
	if value == "" || strings.Contains(value, "/") {
		return 0, errors.New("invalid resource ID")
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid resource ID")
	}

	return id, nil
}

func decodeJSON(r *http.Request, value interface{}) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("request body contains multiple JSON values")
	}

	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
