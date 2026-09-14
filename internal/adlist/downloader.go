package adlist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotModified   = errors.New("adlist not modified")
	ErrResponseLimit = errors.New("adlist response too large")
)

type DownloadResult struct {
	Body         string
	ETag         string
	LastModified string
}

type Downloader struct {
	client    *http.Client
	maxBytes  int64
	userAgent string
}

func NewDownloader() *Downloader {
	return newDownloader(&http.Client{
		Timeout: 15 * time.Second,
	})
}

func newDownloader(client *http.Client) *Downloader {
	maxBytes := int64(64 * 1024 * 1024)

	if value := strings.TrimSpace(os.Getenv("RAMDNS_ADLIST_MAX_BYTES")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			maxBytes = parsed
		}
	}

	return &Downloader{
		client:    client,
		maxBytes:  maxBytes,
		userAgent: "RAMDNS-Adlist/1.0",
	}
}

func (d *Downloader) Download(
	ctx context.Context,
	rawURL string,
	etag string,
	lastModified string,
) (*DownloadResult, error) {
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return nil, errors.New("empty adlist URL")
	}

	parsedURL, err := urlpkg.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid adlist URL: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return nil, errors.New("adlist URL must use HTTPS")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("adlist URL has no host")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		parsedURL.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "text/plain, text/*;q=0.9, */*;q=0.1")

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download adlist: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, ErrNotModified
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected HTTP status: %s",
			resp.Status,
		)
	}

	if resp.ContentLength > d.maxBytes {
		return nil, ErrResponseLimit
	}

	reader := io.LimitReader(
		resp.Body,
		d.maxBytes+1,
	)

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"read adlist response: %w",
			err,
		)
	}

	if int64(len(body)) > d.maxBytes {
		return nil, ErrResponseLimit
	}

	if len(body) == 0 {
		return nil, errors.New("adlist response is empty")
	}

	return &DownloadResult{
		Body:         string(body),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}, nil
}
