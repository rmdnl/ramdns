package cache

import (
	"testing"
	"time"

	"github.com/miekg/dns"
)

func testMessage() *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	msg.Answer = []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: []byte{1, 2, 3, 4},
		},
	}

	return msg
}

func TestCacheSetGet(t *testing.T) {
	c := New(100)

	msg := testMessage()
	key := Key(msg)

	c.Set(key, msg)

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(got.Answer) != 1 {
		t.Fatalf(
			"expected 1 answer, got %d",
			len(got.Answer),
		)
	}

	if got.Answer[0].Header().Ttl > 60 {
		t.Fatalf(
			"TTL increased unexpectedly: %d",
			got.Answer[0].Header().Ttl,
		)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New(100)

	msg := testMessage()
	msg.Answer[0].Header().Ttl = 1

	key := Key(msg)

	c.Set(key, msg)

	if _, ok := c.Get(key); !ok {
		t.Fatal("expected cache hit immediately after Set")
	}

	time.Sleep(1100 * time.Millisecond)

	if _, ok := c.Get(key); ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}
