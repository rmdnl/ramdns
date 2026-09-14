package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDashboardServesIndex(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/index.html", []byte("RAMDNS"), 0600); err != nil {
		t.Fatal(err)
	}

	h := NewHandler("http://127.0.0.1:1", "secret", dir)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "RAMDNS") {
		t.Fatal("dashboard content missing")
	}
}
