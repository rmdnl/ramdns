package filter

import (
	"strings"
	"sync/atomic"
)

type RuleType uint8

const (
	RuleExact RuleType = iota
	RuleDomain
)

type Rule struct {
	Domain string
	Type   RuleType
}

type snapshot struct {
	exact   map[string]struct{}
	domains map[string]struct{}
	size    int
}

type Filter struct {
	current atomic.Pointer[snapshot]
}

func New() *Filter {
	f := &Filter{}

	f.current.Store(&snapshot{
		exact:   make(map[string]struct{}),
		domains: make(map[string]struct{}),
	})

	return f
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	return domain
}

func buildSnapshot(rules []Rule) *snapshot {
	exact := make(map[string]struct{})
	domains := make(map[string]struct{})

	for _, rule := range rules {
		domain := normalizeDomain(rule.Domain)

		if domain == "" {
			continue
		}

		switch rule.Type {
		case RuleExact:
			exact[domain] = struct{}{}

		case RuleDomain:
			domains[domain] = struct{}{}
		}
	}

	return &snapshot{
		exact:   exact,
		domains: domains,
		size:    len(exact) + len(domains),
	}
}

func (f *Filter) Replace(rules []Rule) {
	f.current.Store(buildSnapshot(rules))
}

func (f *Filter) Add(domain string) {
	f.AddExact(domain)
}

func (f *Filter) AddExact(domain string) {
	domain = normalizeDomain(domain)

	if domain == "" {
		return
	}

	current := f.current.Load()

	rules := make([]Rule, 0, current.size+1)

	for domain := range current.exact {
		rules = append(rules, Rule{
			Domain: domain,
			Type:   RuleExact,
		})
	}

	for domain := range current.domains {
		rules = append(rules, Rule{
			Domain: domain,
			Type:   RuleDomain,
		})
	}

	rules = append(rules, Rule{
		Domain: domain,
		Type:   RuleExact,
	})

	f.Replace(rules)
}

func (f *Filter) AddDomain(domain string) {
	domain = normalizeDomain(domain)

	if domain == "" {
		return
	}

	current := f.current.Load()

	rules := make([]Rule, 0, current.size+1)

	for domain := range current.exact {
		rules = append(rules, Rule{
			Domain: domain,
			Type:   RuleExact,
		})
	}

	for domain := range current.domains {
		rules = append(rules, Rule{
			Domain: domain,
			Type:   RuleDomain,
		})
	}

	rules = append(rules, Rule{
		Domain: domain,
		Type:   RuleDomain,
	})

	f.Replace(rules)
}

func (f *Filter) IsBlocked(domain string) bool {
	domain = normalizeDomain(domain)

	if domain == "" {
		return false
	}

	current := f.current.Load()

	if _, ok := current.exact[domain]; ok {
		return true
	}

	for {
		if _, ok := current.domains[domain]; ok {
			return true
		}

		index := strings.IndexByte(domain, '.')
		if index == -1 {
			break
		}

		domain = domain[index+1:]
	}

	return false
}

func (f *Filter) Size() int {
	current := f.current.Load()

	return current.size
}
