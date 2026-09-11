package adlist

import "testing"

func TestParseMixedFormats(t *testing.T) {
	input := `
# comment
! another comment

0.0.0.0 ads.example.com
127.0.0.1 tracker.example.com
||telemetry.example.com^
plain.example.com
`

	parser := NewParser()
	rules := parser.Parse(input)

	if len(rules) != 4 {
		t.Fatalf(
			"expected 4 rules, got %d",
			len(rules),
		)
	}

	expected := []string{
		"ads.example.com",
		"tracker.example.com",
		"telemetry.example.com",
		"plain.example.com",
	}

	for i, rule := range rules {
		if rule.Domain != expected[i] {
			t.Fatalf(
				"rule %d expected %q got %q",
				i,
				expected[i],
				rule.Domain,
			)
		}
	}
}

func TestParseIgnoresInvalidLines(t *testing.T) {
	input := `
# comment
not a domain
http://example.com/list
localhost
127.0.0.1
`

	parser := NewParser()
	rules := parser.Parse(input)

	if len(rules) != 0 {
		t.Fatalf(
			"expected 0 rules, got %d",
			len(rules),
		)
	}
}

func TestParseNormalizesDomains(t *testing.T) {
	input := `
ADS.Example.COM.
||TRACKER.Example.COM^
`

	parser := NewParser()
	rules := parser.Parse(input)

	if len(rules) != 2 {
		t.Fatalf(
			"expected 2 rules, got %d",
			len(rules),
		)
	}

	if rules[0].Domain != "ads.example.com" {
		t.Fatalf(
			"unexpected domain: %s",
			rules[0].Domain,
		)
	}

	if rules[1].Domain != "tracker.example.com" {
		t.Fatalf(
			"unexpected domain: %s",
			rules[1].Domain,
		)
	}
}
