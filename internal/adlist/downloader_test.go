package adlist

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloaderSuccess(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "RAMDNS-Adlist/1.0" {
				t.Fatalf(
					"unexpected User-Agent: %q",
					r.Header.Get("User-Agent"),
				)
			}

			w.Header().Set("ETag", `"abc123"`)
			w.Header().Set(
				"Last-Modified",
				"Fri, 11 Sep 2026 12:00:00 GMT",
			)

			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte(
				"0.0.0.0 ads.example.com\n",
			))
		}),
	)

	defer server.Close()

	d := newDownloader(server.Client())

	result, err := d.Download(
		context.Background(),
		server.URL,
		"",
		"",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Body != "0.0.0.0 ads.example.com\n" {
		t.Fatalf("unexpected body: %q", result.Body)
	}

	if result.ETag != `"abc123"` {
		t.Fatalf("unexpected ETag: %q", result.ETag)
	}

	if result.LastModified == "" {
		t.Fatal("expected Last-Modified header")
	}
}

func TestDownloaderConditionalRequest(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") != `"abc123"` {
				t.Fatalf(
					"unexpected If-None-Match: %q",
					r.Header.Get("If-None-Match"),
				)
			}

			if r.Header.Get("If-Modified-Since") == "" {
				t.Fatal("expected If-Modified-Since")
			}

			w.WriteHeader(http.StatusNotModified)
		}),
	)

	defer server.Close()

	d := newDownloader(server.Client())

	_, err := d.Download(
		context.Background(),
		server.URL,
		`"abc123"`,
		"Fri, 11 Sep 2026 12:00:00 GMT",
	)

	if !errors.Is(err, ErrNotModified) {
		t.Fatalf(
			"expected ErrNotModified, got %v",
			err,
		)
	}
}

func TestDownloaderRejectsHTTP(t *testing.T) {
	d := NewDownloader()

	_, err := d.Download(
		context.Background(),
		"http://example.com/list.txt",
		"",
		"",
	)

	if err == nil {
		t.Fatal("expected HTTP URL to be rejected")
	}
}

func TestDownloaderRejectsHTTPError(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	defer server.Close()

	d := newDownloader(server.Client())

	_, err := d.Download(
		context.Background(),
		server.URL,
		"",
		"",
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestDownloaderRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i := 0; i < 2048; i++ {
				_, _ = w.Write([]byte("x"))
			}
		}),
	)

	defer server.Close()

	d := newDownloader(server.Client())
	d.maxBytes = 1024

	_, err := d.Download(
		context.Background(),
		server.URL,
		"",
		"",
	)

	if !errors.Is(err, ErrResponseLimit) {
		t.Fatalf(
			"expected ErrResponseLimit, got %v",
			err,
		)
	}
}

func TestDownloaderRejectsEmptyResponse(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	defer server.Close()

	d := newDownloader(server.Client())

	_, err := d.Download(
		context.Background(),
		server.URL,
		"",
		"",
	)

	if err == nil {
		t.Fatal("expected empty response error")
	}
}
