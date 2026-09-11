package adlist

import (
	"context"
	"errors"
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

type Manager struct {
	filter     *filter.Filter
	compiler   *Compiler
	downloader *Downloader

	mu     sync.RWMutex
	source SourceState

	customRules []filter.Rule
	adlistRules []filter.Rule
}

func NewManager(f *filter.Filter) *Manager {
	return &Manager{
		filter:     f,
		compiler:   NewCompiler(),
		downloader: NewDownloader(),
	}
}

func (m *Manager) SetCustomRules(rules []filter.Rule) error {
	if len(rules) == 0 {
		return ErrNoRules
	}

	compiled := m.compiler.CompileRules(rules)

	if len(compiled) == 0 {
		return ErrNoRules
	}

	m.mu.Lock()
	m.customRules = compiled
	m.mu.Unlock()

	return m.rebuild()
}

func (m *Manager) Load(lists []string) error {
	if len(lists) == 0 {
		return ErrNoRules
	}

	rules := m.compiler.Compile(lists)

	if len(rules) == 0 {
		return ErrNoRules
	}

	m.mu.Lock()
	m.adlistRules = rules
	m.mu.Unlock()

	return m.rebuild()
}

func (m *Manager) LoadOne(input string) error {
	return m.Load([]string{input})
}

func (m *Manager) LoadURL(
	ctx context.Context,
	url string,
) error {
	m.mu.RLock()
	etag := m.source.ETag
	lastModified := m.source.LastModified
	m.mu.RUnlock()

	result, err := m.downloader.Download(
		ctx,
		url,
		etag,
		lastModified,
	)

	if err != nil {
		if errors.Is(err, ErrNotModified) {
			m.mu.Lock()
			m.source.URL = url
			m.source.LastError = ""
			m.mu.Unlock()

			return nil
		}

		m.mu.Lock()
		m.source.URL = url
		m.source.LastError = err.Error()
		m.mu.Unlock()

		return err
	}

	rules := m.compiler.Compile([]string{result.Body})

	if len(rules) == 0 {
		err := ErrNoRules

		m.mu.Lock()
		m.source.URL = url
		m.source.LastError = err.Error()
		m.mu.Unlock()

		return err
	}

	m.mu.Lock()
	m.adlistRules = rules

	m.source = SourceState{
		URL:          url,
		ETag:         result.ETag,
		LastModified: result.LastModified,
		RuleCount:    len(rules),
		LastSuccess:  time.Now(),
		LastError:    "",
	}
	m.mu.Unlock()

	return m.rebuild()
}

func (m *Manager) rebuild() error {
	m.mu.RLock()

	rules := make(
		[]filter.Rule,
		0,
		len(m.customRules)+len(m.adlistRules),
	)

	rules = append(rules, m.customRules...)
	rules = append(rules, m.adlistRules...)

	m.mu.RUnlock()

	if len(rules) == 0 {
		return ErrNoRules
	}

	compiled := m.compiler.CompileRules(rules)

	if len(compiled) == 0 {
		return ErrNoRules
	}

	m.filter.Replace(compiled)

	return nil
}

func (m *Manager) Validate(lists []string) error {
	if len(lists) == 0 {
		return ErrNoRules
	}

	rules := m.compiler.Compile(lists)

	if len(rules) == 0 {
		return ErrNoRules
	}

	return nil
}

func (m *Manager) RuleCount() int {
	if m.filter == nil {
		return 0
	}

	return m.filter.Size()
}

func (m *Manager) Source() SourceState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.source
}
