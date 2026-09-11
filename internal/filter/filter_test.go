package filter

import "testing"

func TestFilterExactMatch(t *testing.T) {
	f := New()

	f.Add("ads.example.com")

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("expected domain to be blocked")
	}

	if f.IsBlocked("google.com") {
		t.Fatal("google.com should not be blocked")
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	f := New()

	f.Add("Ads.Example.COM")

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("expected lowercase domain to be blocked")
	}

	if !f.IsBlocked("ADS.EXAMPLE.COM") {
		t.Fatal("expected uppercase domain to be blocked")
	}
}

func TestFilterTrailingDot(t *testing.T) {
	f := New()

	f.Add("ads.example.com.")

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("expected domain without dot to be blocked")
	}

	if !f.IsBlocked("ads.example.com.") {
		t.Fatal("expected domain with dot to be blocked")
	}
}

func TestFilterRemove(t *testing.T) {
	f := New()

	f.Add("ads.example.com")

	if !f.IsBlocked("ads.example.com") {
		t.Fatal("expected domain to be blocked")
	}

	f.Remove("ads.example.com")

	if f.IsBlocked("ads.example.com") {
		t.Fatal("expected domain to be unblocked")
	}
}

func TestFilterSize(t *testing.T) {
	f := New()

	f.Add("one.example.com")
	f.Add("two.example.com")
	f.Add("three.example.com")

	if f.Size() != 3 {
		t.Fatalf(
			"expected size 3, got %d",
			f.Size(),
		)
	}
}
