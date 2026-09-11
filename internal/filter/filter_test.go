package filter

import (
	"sync"
	"testing"
)

func TestFilterExactMatch(t *testing.T) {
	f := New()

	f.AddExact("ads.example.com")

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("expected exact domain to be blocked")
	}

	if f.IsBlocked("foo.ads.example.com") {
		t.Fatal("subdomain should not be blocked by exact rule")
	}

	if f.IsBlocked("example.com") {
		t.Fatal("parent domain should not be blocked")
	}
}

func TestFilterDomainMatch(t *testing.T) {
	f := New()

	f.AddDomain("example.com")

	tests := []struct {
		domain  string
		blocked bool
	}{
		{"example.com", true},
		{"www.example.com", true},
		{"ads.example.com", true},
		{"foo.ads.example.com", true},
		{"example.net", false},
		{"notexample.com", false},
	}

	for _, test := range tests {
		got := f.IsBlocked(test.domain)

		if got != test.blocked {
			t.Fatalf(
				"domain=%q expected=%v got=%v",
				test.domain,
				test.blocked,
				got,
			)
		}
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	f := New()

	f.AddDomain("Ads.Example.COM")

	if !f.IsBlocked("ADS.EXAMPLE.COM") {
		t.Fatal("expected uppercase domain to be blocked")
	}

	if !f.IsBlocked("foo.ads.example.com.") {
		t.Fatal("expected trailing-dot domain to be blocked")
	}
}

func TestFilterReplace(t *testing.T) {
	f := New()

	f.AddDomain("old.example.com")

	if !f.IsBlocked("old.example.com") {
		t.Fatal("expected old rule to work")
	}

	f.Replace([]Rule{
		{
			Domain: "new.example.com",
			Type:   RuleDomain,
		},
	})

	if f.IsBlocked("old.example.com") {
		t.Fatal("old rule should no longer exist")
	}

	if !f.IsBlocked("new.example.com") {
		t.Fatal("new rule should exist")
	}
}

func TestFilterSize(t *testing.T) {
	f := New()

	f.Replace([]Rule{
		{
			Domain: "one.example.com",
			Type:   RuleExact,
		},
		{
			Domain: "two.example.com",
			Type:   RuleDomain,
		},
		{
			Domain: "three.example.com",
			Type:   RuleExact,
		},
	})

	if f.Size() != 3 {
		t.Fatalf(
			"expected size 3, got %d",
			f.Size(),
		)
	}
}

func TestFilterConcurrentLookup(t *testing.T) {
	f := New()

	f.Replace([]Rule{
		{
			Domain: "example.com",
			Type:   RuleDomain,
		},
	})

	const workers = 50
	const iterations = 1000

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				if !f.IsBlocked("ads.example.com") {
					t.Error("expected domain to be blocked")
					return
				}
			}
		}()
	}

	wg.Wait()
}
