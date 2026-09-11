package adlist

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ramdns/ramdns/internal/filter"
)

var (
	ErrNoRules = errors.New("adlist produced no valid rules")
)

type SourceState struct {
	URL          string
	ETag         string
	LastModified string
	RuleCount    int
	LastSuccess  time.Time
	LastError    string
}

type sourceCache struct {
	state SourceState
	body  string
}

type Manager struct {
	filter     *filter.Filter
	compiler   *Compiler
	downloader *Downloader

	mu sync.RWMutex

	source  SourceState
	sources map[string]SourceState
	cache   map[string]sourceCache

	customRules []filter.Rule
	adlistRules []filter.Rule
}

func NewManager(f *filter.Filter) *Manager {
	return &Manager{
		filter:     f,
		compiler:   NewCompiler(),
		downloader: NewDownloader(),
		sources:    make(map[string]SourceState),
		cache:      make(map[string]sourceCache),
	}
}

// LoadOne keeps the original API semantics.
// The argument is raw adlist content, not a URL.
func (m *Manager) LoadOne(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrNoRules
	}

	rules := m.compiler.Compile([]string{content})
	if len(rules) == 0 {
		return ErrNoRules
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.adlistRules = rules

	// Preserve legacy single-source metadata behavior.
	m.source.RuleCount = len(rules)
	m.source.LastSuccess = time.Now()
	m.source.LastError = ""

	return m.rebuildLocked()
}

// LoadURL downloads and loads one adlist URL.
func (m *Manager) LoadURL(ctx context.Context, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("empty adlist URL")
	}

	return m.loadURLs(ctx, []string{rawURL})
}

// LoadURLs downloads multiple adlist URLs concurrently.
// The active snapshot is replaced only after every source succeeds.
func (m *Manager) LoadURLs(ctx context.Context, urls []string) error {
	return m.loadURLs(ctx, urls)
}

func (m *Manager) loadURLs(ctx context.Context, urls []string) error {
	cleanURLs := normalizeURLs(urls)
	if len(cleanURLs) == 0 {
		return errors.New("no adlist URLs configured")
	}

	type result struct {
		url          string
		body         string
		etag         string
		lastModified string
		notModified  bool
		err          error
	}

	results := make(chan result, len(cleanURLs))

	var wg sync.WaitGroup
	wg.Add(len(cleanURLs))

	for _, url := range cleanURLs {
		url := url

		go func() {
			defer wg.Done()

			m.mu.RLock()

			cached, hasCache := m.cache[url]
			legacy := m.source

			m.mu.RUnlock()

			var etag string
			var lastModified string

			if hasCache {
				etag = cached.state.ETag
				lastModified = cached.state.LastModified
			}

			// Legacy compatibility:
			// existing callers/tests may seed manager.source metadata.
			if etag == "" {
				etag = legacy.ETag
			}
			if lastModified == "" {
				lastModified = legacy.LastModified
			}

			download, err := m.downloader.Download(
				ctx,
				url,
				etag,
				lastModified,
			)

			if err != nil {
				if errors.Is(err, ErrNotModified) {
					if hasCache && strings.TrimSpace(cached.body) != "" {
						results <- result{
							url:          url,
							body:         cached.body,
							etag:         cached.state.ETag,
							lastModified: cached.state.LastModified,
							notModified:  true,
						}
						return
					}

					// If there is no body cache, the existing active
					// snapshot can remain active. Returning ErrNotModified
					// preserves the original manager behavior.
					results <- result{
						url:         url,
						notModified: true,
						err:         ErrNotModified,
					}
					return
				}

				results <- result{
					url: url,
					err: fmt.Errorf("download adlist: %w", err),
				}
				return
			}

			results <- result{
				url:          url,
				body:         download.Body,
				etag:         download.ETag,
				lastModified: download.LastModified,
			}
		}()
	}

	wg.Wait()
	close(results)

	bodies := make(map[string]string, len(cleanURLs))
	newSources := make(map[string]SourceState, len(cleanURLs))

	for res := range results {
		if res.err != nil {
			if errors.Is(res.err, ErrNotModified) {
				// A 304 means the currently active snapshot is still valid.
				continue
			}

			m.mu.Lock()

			m.source.LastError = res.err.Error()

			if old, ok := m.sources[res.url]; ok {
				old.LastError = res.err.Error()
				m.sources[res.url] = old
			}

			m.mu.Unlock()

			// Never replace the active filter on failed update.
			return res.err
		}

		rules := m.compiler.Compile([]string{res.body})
		if len(rules) == 0 {
			err := fmt.Errorf(
				"adlist %s produced no valid rules",
				res.url,
			)

			m.mu.Lock()
			m.source.LastError = err.Error()
			m.mu.Unlock()

			return err
		}

		now := time.Now()

		state := SourceState{
			URL:          res.url,
			ETag:         res.etag,
			LastModified: res.lastModified,
			RuleCount:    len(rules),
			LastSuccess:  now,
		}

		if res.notModified {
			m.mu.RLock()

			if old, ok := m.sources[res.url]; ok {
				state = old
				state.LastSuccess = now
				state.LastError = ""
			} else if m.source.URL == res.url || len(cleanURLs) == 1 {
				state = m.source
				state.URL = res.url
				state.LastSuccess = now
				state.LastError = ""
			}

			m.mu.RUnlock()
		}

		bodies[res.url] = res.body
		newSources[res.url] = state
	}

	// If every source returned 304 and there is no new body,
	// leave the current active snapshot untouched.
	if len(bodies) == 0 {
		return ErrNotModified
	}

	var allRules []filter.Rule

	for _, url := range cleanURLs {
		body, ok := bodies[url]
		if !ok {
			// Source was 304. Reuse cached body if available.
			m.mu.RLock()
			cached, cachedOK := m.cache[url]
			m.mu.RUnlock()

			if !cachedOK || strings.TrimSpace(cached.body) == "" {
				return fmt.Errorf("no cached body for unchanged adlist %s", url)
			}

			body = cached.body
		}

		rules := m.compiler.Compile([]string{body})
		allRules = append(allRules, rules...)
	}

	if len(allRules) == 0 {
		return ErrNoRules
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.adlistRules = allRules

	// Preserve per-source state.
	for url, state := range newSources {
		m.sources[url] = state

		if body, ok := bodies[url]; ok {
			m.cache[url] = sourceCache{
				state: state,
				body:  body,
			}
		}
	}

	if len(cleanURLs) == 1 {
		state := newSources[cleanURLs[0]]

		// If source returned 304 and was only available through legacy
		// metadata, preserve the existing ETag/Last-Modified.
		if state.ETag == "" {
			state.ETag = m.source.ETag
		}
		if state.LastModified == "" {
			state.LastModified = m.source.LastModified
		}

		state.URL = cleanURLs[0]
		state.RuleCount = len(allRules)
		state.LastSuccess = time.Now()
		state.LastError = ""

		m.source = state
	} else {
		m.source = SourceState{
			URL:         strings.Join(cleanURLs, ","),
			RuleCount:   len(allRules),
			LastSuccess: time.Now(),
		}
	}

	return m.rebuildLocked()
}

func (m *Manager) RuleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.adlistRules)
}

func (m *Manager) Source() SourceState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.source
}

func (m *Manager) Sources() []SourceState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]SourceState, 0, len(m.sources))

	for _, state := range m.sources {
		result = append(result, state)
	}

	return result
}

func (m *Manager) rebuild() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.rebuildLocked()
}

func (m *Manager) rebuildLocked() error {
	rules := make([]filter.Rule, 0, len(m.customRules)+len(m.adlistRules))

	rules = append(rules, m.customRules...)
	rules = append(rules, m.adlistRules...)

	m.filter.Replace(rules)

	return nil
}

func normalizeURLs(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	result := make([]string, 0, len(urls))

	for _, rawURL := range urls {
		url := strings.TrimSpace(rawURL)

		if url == "" {
			continue
		}

		if _, exists := seen[url]; exists {
			continue
		}

		seen[url] = struct{}{}
		result = append(result, url)
	}

	return result
}
