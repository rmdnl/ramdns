package adlist

import (
	"testing"

	"github.com/ramdns/ramdns/internal/filter"
)

func TestCompilerDeduplicates(t *testing.T) {
	lists := []string{
		`
ads.example.com
tracker.example.com
`,
		`
ads.example.com
telemetry.example.com
tracker.example.com
`,
	}

	compiler := NewCompiler()
	rules := compiler.Compile(lists)

	if len(rules) != 3 {
		t.Fatalf(
			"expected 3 unique rules, got %d",
			len(rules),
		)
	}
}

func TestCompilerPrefersDomainRule(t *testing.T) {
	lists := []string{
		`
ads.example.com
`,
		`
||ads.example.com^
`,
	}

	compiler := NewCompiler()
	rules := compiler.Compile(lists)

	if len(rules) != 1 {
		t.Fatalf(
			"expected 1 rule, got %d",
			len(rules),
		)
	}

	if rules[0].Domain != "ads.example.com" {
		t.Fatalf(
			"unexpected domain: %s",
			rules[0].Domain,
		)
	}

	if rules[0].Type != filter.RuleDomain {
		t.Fatalf(
			"expected RuleDomain, got %v",
			rules[0].Type,
		)
	}
}

func TestCompilerWorksWithMultipleLists(t *testing.T) {
	lists := []string{
		`
# list one
0.0.0.0 ads.one.com
0.0.0.0 tracker.one.com
`,
		`
# list two
||ads.two.com^
plain.two.com
`,
		`
# list three
ads.one.com
tracker.three.com
`,
	}

	compiler := NewCompiler()
	rules := compiler.Compile(lists)

	if len(rules) != 5 {
		t.Fatalf(
			"expected 5 unique rules, got %d",
			len(rules),
		)
	}
}
