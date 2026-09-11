package adlist

import (
	"bufio"
	"errors"
	"net"
	"strings"

	"github.com/ramdns/ramdns/internal/filter"
)

var (
	ErrEmptyList = errors.New("adlist is empty")
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(input string) []filter.Rule {
	scanner := bufio.NewScanner(strings.NewReader(input))

	rules := make([]filter.Rule, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "!") {
			continue
		}

		domain, ok := parseLine(line)

		if !ok {
			continue
		}

		rules = append(rules, filter.Rule{
			Domain: domain,
			Type:   filter.RuleDomain,
		})
	}

	return rules
}

func parseLine(line string) (string, bool) {
	line = strings.TrimSpace(line)

	// Hosts format:
	// 0.0.0.0 example.com
	// 127.0.0.1 example.com
	fields := strings.Fields(line)

	if len(fields) >= 2 {
		ip := fields[0]

		if ip == "0.0.0.0" ||
			ip == "127.0.0.1" ||
			ip == "::" ||
			ip == "::1" {
			return normalize(fields[1])
		}
	}

	// Adblock domain format:
	// ||example.com^
	if strings.HasPrefix(line, "||") {
		domain := strings.TrimPrefix(line, "||")

		if index := strings.IndexByte(domain, '^'); index >= 0 {
			domain = domain[:index]
		}

		return normalize(domain)
	}

	// Plain domain:
	// example.com
	if !strings.ContainsAny(line, " /\\") {
		return normalize(line)
	}

	return "", false
}

func normalize(domain string) (string, bool) {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	if domain == "" {
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
		if part == "" {
			return "", false
		}

		for _, ch := range part {
			if (ch >= 'a' && ch <= 'z') ||
				(ch >= '0' && ch <= '9') ||
				ch == '-' ||
				ch == '_' {
				continue
			}

			return "", false
		}
	}

	return domain, true
}
