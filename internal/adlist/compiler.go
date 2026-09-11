package adlist

import (
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

func (c *Compiler) Compile(lists []string) []filter.Rule {
	allRules := make([]filter.Rule, 0)

	for _, input := range lists {
		rules := c.parser.Parse(input)
		allRules = append(allRules, rules...)
	}

	return c.CompileRules(allRules)
}

func (c *Compiler) CompileRules(
	rules []filter.Rule,
) []filter.Rule {
	unique := make(map[string]filter.Rule)

	for _, rule := range rules {
		key := rule.Domain

		if key == "" {
			continue
		}

		if existing, ok := unique[key]; ok {
			if existing.Type == filter.RuleDomain {
				continue
			}
		}

		unique[key] = rule
	}

	result := make([]filter.Rule, 0, len(unique))

	for _, rule := range unique {
		result = append(result, rule)
	}

	return result
}
