package management

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramdns/ramdns/internal/metrics"
)

func TestAuthorization(t *testing.T) {
	h := &Handler{
		metrics: &metrics.Metrics{},
		token:   "test-secret",
	}

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic test-secret", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"valid token", "Bearer test-secret", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
			req.Header.Set("Authorization", tt.header)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("got status %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestUnknownEndpoint(t *testing.T) {
	h := &Handler{
		metrics: &metrics.Metrics{},
		token:   "test-secret",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil)
	req.Header.Set("Authorization", "Bearer test-secret")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}
