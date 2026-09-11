package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Entry struct {
	Msg       *dns.Msg
	ExpiresAt time.Time
}

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]Entry
	maxEntries int
}

func New(maxEntries int) *Cache {
	return &Cache{
		entries:    make(map[string]Entry, maxEntries),
		maxEntries: maxEntries,
	}
}

func Key(req *dns.Msg) string {
	if len(req.Question) == 0 {
		return ""
	}

	q := req.Question[0]

	do := false
	if edns := req.IsEdns0(); edns != nil {
		do = edns.Do()
	}

	return fmt.Sprintf(
		"%s|%d|%d|do=%t",
		strings.ToLower(dns.Fqdn(q.Name)),
		q.Qtype,
		q.Qclass,
		do,
	)
}

func (c *Cache) Get(key string) (*dns.Msg, bool) {
	if key == "" {
		return nil, false
	}

	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if !now.Before(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()

		return nil, false
	}

	msg := entry.Msg.Copy()

	remaining := uint32(entry.ExpiresAt.Sub(now).Seconds())
	setTTL(msg, remaining)

	return msg, true
}

func (c *Cache) Set(key string, msg *dns.Msg) {
	if key == "" || msg == nil {
		return
	}

	ttl, ok := minTTL(msg)
	if !ok || ttl == 0 {
		return
	}

	entry := Entry{
		Msg:       msg.Copy(),
		ExpiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxEntries {
		c.evictExpired()
	}

	if len(c.entries) >= c.maxEntries {
		c.evictOne()
	}

	c.entries[key] = entry
}

func (c *Cache) evictExpired() {
	now := time.Now()

	for key, entry := range c.entries {
		if !now.Before(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) evictOne() {
	var oldestKey string
	var oldestExpiry time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestExpiry) {
			oldestKey = key
			oldestExpiry = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

func minTTL(msg *dns.Msg) (uint32, bool) {
	var min uint32
	found := false

	for _, rr := range msg.Answer {
		ttl := rr.Header().Ttl

		if !found || ttl < min {
			min = ttl
			found = true
		}
	}

	for _, rr := range msg.Ns {
		ttl := rr.Header().Ttl

		if !found || ttl < min {
			min = ttl
			found = true
		}
	}

	for _, rr := range msg.Extra {
		if rr.Header().Rrtype == dns.TypeOPT {
			continue
		}

		ttl := rr.Header().Ttl

		if !found || ttl < min {
			min = ttl
			found = true
		}
	}

	return min, found
}

func setTTL(msg *dns.Msg, ttl uint32) {
	for _, rr := range msg.Answer {
		rr.Header().Ttl = ttl
	}

	for _, rr := range msg.Ns {
		rr.Header().Ttl = ttl
	}

	for _, rr := range msg.Extra {
		if rr.Header().Rrtype == dns.TypeOPT {
			continue
		}

		rr.Header().Ttl = ttl
	}
}
