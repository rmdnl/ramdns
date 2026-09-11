package upstream

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestNew(t *testing.T) {
	r, err := New([]string{
		"1.1.1.1:853|cloudflare-dns.com",
		"8.8.8.8:853|dns.google",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(r.Servers))
	}

	if r.Servers[0].Addr != "1.1.1.1:853" {
		t.Fatalf("unexpected primary address")
	}

	if r.Servers[0].ServerName != "cloudflare-dns.com" {
		t.Fatalf("unexpected primary server name")
	}

	if r.Servers[1].Addr != "8.8.8.8:853" {
		t.Fatalf("unexpected backup address")
	}

	if r.Servers[1].ServerName != "dns.google" {
		t.Fatalf("unexpected backup server name")
	}

	if r.Servers[0].Client.Net != "tcp-tls" {
		t.Fatalf("expected tcp-tls client")
	}

	if r.Servers[0].Client.TLSConfig == nil {
		t.Fatal("expected TLS config")
	}

	if r.Servers[0].Client.TLSConfig.MinVersion != 0x0304 {
		t.Fatalf("expected TLS 1.3 minimum")
	}

	if r.Servers[0].Client.TLSConfig.InsecureSkipVerify {
		t.Fatal("TLS verification must remain enabled")
	}
}

func TestNewInvalid(t *testing.T) {
	_, err := New([]string{
		"invalid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeContextTimeout(t *testing.T) {
	r, err := New([]string{
		"192.0.2.1:853|primary.invalid",
		"192.0.2.2:853|backup.invalid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, server := range r.Servers {
		server.Client.Timeout = 100 * time.Millisecond
	}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	_, err = r.Exchange(ctx, req)
	if err == nil {
		t.Fatal("expected upstream failure")
	}
}

func TestExchangeNilRequest(t *testing.T) {
	r, err := New([]string{
		"192.0.2.1:853|primary.invalid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = r.Exchange(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil request error")
	}
}

func TestExchangeNoServers(t *testing.T) {
	r := &Resolver{}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)

	_, err := r.Exchange(context.Background(), req)
	if err == nil {
		t.Fatal("expected no upstream servers error")
	}
}

func TestExchangeParallelTimeoutBound(t *testing.T) {
	r, err := New([]string{
		"192.0.2.1:853|primary.invalid",
		"192.0.2.2:853|backup.invalid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	start := time.Now()
	_, err = r.Exchange(ctx, req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected upstream failure")
	}

	if elapsed > 2*time.Second {
		t.Fatalf(
			"exchange exceeded context deadline: %s",
			elapsed,
		)
	}
}
