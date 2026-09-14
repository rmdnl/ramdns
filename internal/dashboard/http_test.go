package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestDashboardDoesNotExposeManagementTokenToFrontend(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "dashboard", "index.html"))
	if err != nil {
		t.Fatalf("read dashboard: %v", err)
	}

	html := string(data)

	for _, forbidden := range []string{
		"sessionStorage",
		"localStorage",
		"ramdns_token",
		"Management token",
		"Authorization: \"Bearer \"",
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("dashboard frontend contains forbidden credential reference %q", forbidden)
		}
	}
}
