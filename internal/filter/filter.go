package filter

import (
	"strings"
	"sync"
)

type Filter struct {
	mu      sync.RWMutex
	domains map[string]struct{}
}

func New() *Filter {
	return &Filter{
		domains: make(map[string]struct{}),
	}
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	return domain
}

func (f *Filter) Add(domain string) {
	domain = normalizeDomain(domain)

	if domain == "" {
		return
	}

	f.mu.Lock()
	f.domains[domain] = struct{}{}
	f.mu.Unlock()
}

func (f *Filter) Remove(domain string) {
	domain = normalizeDomain(domain)

	if domain == "" {
		return
	}

	f.mu.Lock()
	delete(f.domains, domain)
	f.mu.Unlock()
}

func (f *Filter) IsBlocked(domain string) bool {
	domain = normalizeDomain(domain)

	if domain == "" {
		return false
	}

	f.mu.RLock()
	_, blocked := f.domains[domain]
	f.mu.RUnlock()

	return blocked
}

func (f *Filter) Size() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.domains)
}
