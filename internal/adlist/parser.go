package adlist

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/ramdns/ramdns/internal/filter"
)

var (
	ErrEmptyList = errors.New("adlist is empty")
)

const maxLineSize = 1024 * 1024

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(input string) []filter.Rule {
	rules := make([]filter.Rule, 0)
	p.parseReader(strings.NewReader(input), func(rule filter.Rule) {
		rules = append(rules, rule)
	})
	return rules
}

func (p *Parser) parseReader(r io.Reader, emit func(filter.Rule)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		domain, ruleType, ok := parseLine(line)
		if !ok {
			continue
		}

		emit(filter.Rule{
			Domain: domain,
			Type:   ruleType,
		})
	}
}

func parseLine(line string) (string, filter.RuleType, bool) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))

	if line == "" {
		return "", 0, false
	}

	// Comments and list metadata.
	if strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, "!") ||
		strings.HasPrefix(line, ";") ||
		strings.HasPrefix(line, "[Adblock") ||
		strings.HasPrefix(line, "[uBlock") {
		return "", 0, false
	}

	// Explicit exception/allow rules cannot be represented by the
	// current blocking engine, so never accidentally turn them into blocks.
	if strings.HasPrefix(line, "@@") {
		return "", 0, false
	}

	// Hosts files:
	// 0.0.0.0 example.com
	// 127.0.0.1 example.com
	// :: example.com
	// ::1 example.com
	if fields := strings.Fields(line); len(fields) >= 2 {
		switch fields[0] {
		case "0.0.0.0", "127.0.0.1", "::", "::1":
			domain := fields[1]
			if d, ok := normalize(domain); ok {
				return d, filter.RuleDomain, true
			}
			return "", 0, false
		}
	}

	// dnsmasq:
	// address=/example.com/0.0.0.0
	// address=/example.com/
	if strings.HasPrefix(line, "address=/") {
		value := strings.TrimPrefix(line, "address=/")
		if index := strings.IndexByte(value, '/'); index >= 0 {
			value = value[:index]
		}
		if d, ok := normalizeWildcard(value); ok {
			return d, filter.RuleDomain, true
		}
		return "", 0, false
	}

	// dnsmasq server=/domain/server
	// This is routing, not a block rule, so do not interpret it as one.
	if strings.HasPrefix(line, "server=/") {
		return "", 0, false
	}

	// Adblock/uBlock:
	// ||example.com^
	// ||example.com^$script,image
	// ||*.example.com^
	if strings.HasPrefix(line, "||") {
		value := strings.TrimPrefix(line, "||")

		if index := strings.IndexByte(value, '^'); index >= 0 {
			value = value[:index]
		}

		if index := strings.IndexByte(value, '$'); index >= 0 {
			value = value[:index]
		}

		if d, ok := normalizeWildcard(value); ok {
			return d, filter.RuleDomain, true
		}
		return "", 0, false
	}

	// Hosts-style lines with inline comments.
	// example.com # comment
	if index := strings.IndexByte(line, '#'); index >= 0 {
		line = strings.TrimSpace(line[:index])
	}

	// Plain wildcard:
	// *.example.com
	if d, ok := normalizeWildcard(line); ok {
		return d, filter.RuleDomain, true
	}

	return "", 0, false
}

func normalizeWildcard(domain string) (string, bool) {
	domain = strings.TrimSpace(domain)

	for strings.HasPrefix(domain, "*.") {
		domain = strings.TrimPrefix(domain, "*.")
	}

	return normalize(domain)
}

func normalize(domain string) (string, bool) {
	domain = strings.TrimSpace(domain)
	domain = strings.Trim(domain, ".")
	domain = strings.ToLower(domain)

	if domain == "" || len(domain) > 253 {
		return "", false
	}

	if net.ParseIP(domain) != nil {
		return "", false
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", false
	}

	for _, part := range parts {
		if part == "" || len(part) > 63 {
			return "", false
		}

		if part[0] == '-' || part[len(part)-1] == '-' {
			return "", false
		}

		for _, ch := range part {
			if (ch >= 'a' && ch <= 'z') ||
				(ch >= '0' && ch <= '9') ||
				ch == '-' {
				continue
			}

			// DNS IDN should arrive as punycode in adlists.
			if ch >= 128 {
				return "", false
			}

			return "", false
		}
	}

	return domain, true
}
