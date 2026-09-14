package adlist

import (
	"strings"

	"github.com/ramdns/ramdns/internal/filter"
)

type Compiler struct {
	parser *Parser
}

func NewCompiler() *Compiler {
	return &Compiler{
		parser: NewParser(),
	}
}

// Compile parses and deduplicates one or more adlists.
//
// Deduplication happens while compiling so callers receive a compact
// rule set without retaining duplicate rules from the source lists.
func (c *Compiler) Compile(lists []string) []filter.Rule {
	if len(lists) == 0 {
		return nil
	}

	unique := make(map[string]filter.Rule)

	for _, input := range lists {
		c.parser.parseReader(strings.NewReader(input), func(rule filter.Rule) {
			if rule.Domain == "" {
				return
			}

			existing, exists := unique[rule.Domain]
			if exists && existing.Type == filter.RuleDomain {
				return
			}

			unique[rule.Domain] = rule
		})
	}

	return rulesFromMap(unique)
}

// CompileRules deduplicates an already parsed rule set.
func (c *Compiler) CompileRules(rules []filter.Rule) []filter.Rule {
	if len(rules) == 0 {
		return nil
	}

	unique := make(map[string]filter.Rule, len(rules))

	for _, rule := range rules {
		if rule.Domain == "" {
			continue
		}

		existing, exists := unique[rule.Domain]
		if exists && existing.Type == filter.RuleDomain {
			continue
		}

		unique[rule.Domain] = rule
	}

	return rulesFromMap(unique)
}

func rulesFromMap(unique map[string]filter.Rule) []filter.Rule {
	if len(unique) == 0 {
		return nil
	}

	result := make([]filter.Rule, 0, len(unique))
	for _, rule := range unique {
		result = append(result, rule)
	}

	return result
}
