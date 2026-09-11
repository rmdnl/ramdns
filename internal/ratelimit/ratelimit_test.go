package ratelimit

import (
	"net"
	"testing"
	"time"
)

func TestAllowBurst(t *testing.T) {
	l := New(1, 3, 100)

	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.0.2.1"),
		Port: 12345,
	}

	for i := 0; i < 3; i++ {
		if !l.Allow(addr) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if l.Allow(addr) {
		t.Fatal("request beyond burst should be rejected")
	}
}

func TestRefill(t *testing.T) {
	l := New(100, 1, 100)

	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.0.2.2"),
		Port: 12345,
	}

	if !l.Allow(addr) {
		t.Fatal("first request should be allowed")
	}

	if l.Allow(addr) {
		t.Fatal("second immediate request should be rejected")
	}

	time.Sleep(20 * time.Millisecond)

	if !l.Allow(addr) {
		t.Fatal("token should have refilled")
	}
}

func TestDifferentIPs(t *testing.T) {
	l := New(1, 1, 100)

	addr1 := &net.UDPAddr{
		IP:   net.ParseIP("192.0.2.10"),
		Port: 12345,
	}

	addr2 := &net.UDPAddr{
		IP:   net.ParseIP("192.0.2.11"),
		Port: 12345,
	}

	if !l.Allow(addr1) {
		t.Fatal("first IP should be allowed")
	}

	if !l.Allow(addr2) {
		t.Fatal("second IP should be allowed")
	}
}
