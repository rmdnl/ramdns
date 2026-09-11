package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	m := &Metrics{}

	m.CacheHits.Add(8)
	m.CacheMisses.Add(2)
	m.UpstreamQuery.Add(2)

	handler := NewHandler(m)

	req := httptest.NewRequest(
		http.MethodGet,
		"/metrics",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			rec.Code,
		)
	}

	body := rec.Body.String()

	if body == "" {
		t.Fatal("expected metrics response")
	}
}

func TestMetricsHandlerNotFound(t *testing.T) {
	m := &Metrics{}
	handler := NewHandler(m)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status 404, got %d",
			rec.Code,
		)
	}
}
