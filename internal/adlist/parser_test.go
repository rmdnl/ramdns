package adlist

import (
	"reflect"
	"testing"

	"github.com/ramdns/ramdns/internal/filter"
)

func TestParserFormats(t *testing.T) {
	input := `
! comment
# comment
; comment
[Adblock Plus 2.0]

||ads.example.com^
||tracker.example.net^$script,image,third-party
||*.wild.example.org^

0.0.0.0 hosts.example.com
127.0.0.1 localhost.example.net
:: ipv6.example.org
::1 ipv6loop.example.org

address=/dnsmasq.example.com/0.0.0.0

https://url.example.com/path/banner.js
http://tracker.example.net/foo

plain.example.org
*.wildcard.example.com
plain.example.org.

@@||allowed.example.com^
server=/not-a-block.example/1.1.1.1

192.168.1.1
invalid
bad domain.example.com
https://
`

	got := NewParser().Parse(input)

	want := []filter.Rule{
		{Domain: "ads.example.com", Type: filter.RuleDomain},
		{Domain: "tracker.example.net", Type: filter.RuleDomain},
		{Domain: "wild.example.org", Type: filter.RuleDomain},
		{Domain: "hosts.example.com", Type: filter.RuleDomain},
		{Domain: "localhost.example.net", Type: filter.RuleDomain},
		{Domain: "ipv6.example.org", Type: filter.RuleDomain},
		{Domain: "ipv6loop.example.org", Type: filter.RuleDomain},
		{Domain: "dnsmasq.example.com", Type: filter.RuleDomain},

		{Domain: "plain.example.org", Type: filter.RuleDomain},
		{Domain: "wildcard.example.com", Type: filter.RuleDomain},
		{Domain: "plain.example.org", Type: filter.RuleDomain},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected rules:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParserRejectsExceptionsAndInvalidInput(t *testing.T) {
	input := `
@@||allowed.example.com^
192.168.1.1
127.0.0.1
invalid
-evil.example.com
evil-.example.com
foo..example.com
https://
`

	got := NewParser().Parse(input)

	if len(got) != 0 {
		t.Fatalf("expected no rules, got %#v", got)
	}
}

func TestParserLargeLineDoesNotCrash(t *testing.T) {
	line := make([]byte, maxLineSize+1)
	for i := range line {
		line[i] = 'a'
	}

	got := NewParser().Parse(string(line))
	if len(got) != 0 {
		t.Fatalf("expected oversized line to be ignored")
	}
}

func TestParserIgnoresSourceURLs(t *testing.T) {
	input := `
http://example.com/list
https://example.org/hosts.txt
ftp://example.net/blocklist
`

	rules := NewParser().Parse(input)
	if len(rules) != 0 {
		t.Fatalf("expected source URLs to be ignored, got %d rules", len(rules))
	}
}

func BenchmarkParser10K(b *testing.B) {
	input := makeBenchmarkAdlist(10_000)
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(input)
	}
}

func BenchmarkParser100K(b *testing.B) {
	input := makeBenchmarkAdlist(100_000)
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(input)
	}
}
