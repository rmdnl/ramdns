package adlist

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramdns/ramdns/internal/filter"
)

func TestManagerLoad(t *testing.T) {
	f := filter.New()
	manager := NewManager(f)

	input := `
0.0.0.0 ads.example.com
||tracker.example.com^
`

	if err := manager.LoadOne(input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.RuleCount() != 2 {
		t.Fatalf(
			"expected 2 rules, got %d",
			manager.RuleCount(),
		)
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("ads.example.com should be blocked")
	}

	if !f.IsBlocked("foo.tracker.example.com") {
		t.Fatal("foo.tracker.example.com should be blocked")
	}
}

func TestManagerRejectsEmptyList(t *testing.T) {
	f := filter.New()
	manager := NewManager(f)

	if err := manager.LoadOne(""); err == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestManagerKeepsOldSnapshotOnFailure(t *testing.T) {
	f := filter.New()
	manager := NewManager(f)

	if err := manager.LoadOne(`
ads.example.com
`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("initial rule should be active")
	}

	err := manager.LoadOne(`
# invalid list
localhost
127.0.0.1
http://example.com/list
`)

	if err == nil {
		t.Fatal("expected second load to fail")
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("old snapshot was unexpectedly replaced")
	}
}

func TestManagerLoadURL(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ETag", `"v1"`)

			_, _ = w.Write([]byte(`
0.0.0.0 ads.example.com
||tracker.example.com^
`))
		}),
	)

	defer server.Close()

	f := filter.New()
	manager := NewManager(f)

	// Inject test HTTP client.
	manager.downloader = newDownloader(server.Client())

	err := manager.LoadURL(
		context.Background(),
		server.URL,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("ads.example.com should be blocked")
	}

	if !f.IsBlocked("foo.tracker.example.com") {
		t.Fatal("foo.tracker.example.com should be blocked")
	}

	source := manager.Source()

	if source.ETag != `"v1"` {
		t.Fatalf(
			"expected ETag %q, got %q",
			`"v1"`,
			source.ETag,
		)
	}

	if source.RuleCount != 2 {
		t.Fatalf(
			"expected 2 rules, got %d",
			source.RuleCount,
		)
	}

	if source.LastSuccess.IsZero() {
		t.Fatal("expected LastSuccess")
	}
}

func TestManagerLoadURLFailureKeepsSnapshot(t *testing.T) {
	f := filter.New()
	manager := NewManager(f)

	if err := manager.LoadOne(`
ads.example.com
`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	defer server.Close()

	manager.downloader = newDownloader(server.Client())

	err := manager.LoadURL(
		context.Background(),
		server.URL,
	)

	if err == nil {
		t.Fatal("expected download error")
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("old snapshot was replaced after download failure")
	}

	source := manager.Source()

	if source.LastError == "" {
		t.Fatal("expected LastError")
	}
}

func TestManagerLoadURLNotModified(t *testing.T) {
	f := filter.New()
	manager := NewManager(f)

	if err := manager.LoadOne(`
ads.example.com
`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") != `"v1"` {
				t.Fatalf(
					"expected If-None-Match %q, got %q",
					`"v1"`,
					r.Header.Get("If-None-Match"),
				)
			}

			w.WriteHeader(http.StatusNotModified)
		}),
	)

	defer server.Close()

	manager.downloader = newDownloader(server.Client())

	// Seed metadata.
	manager.mu.Lock()
	manager.source.ETag = `"v1"`
	manager.mu.Unlock()

	err := manager.LoadURL(
		context.Background(),
		server.URL,
	)

	if err != nil && !errors.Is(err, ErrNotModified) {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("snapshot should remain active after 304")
	}
}
