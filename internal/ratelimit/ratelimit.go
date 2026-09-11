package ratelimit

import (
	"net"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
	seen   time.Time
}

type Limiter struct {
	mu sync.Mutex

	rate    float64
	burst   float64
	maxIPs  int
	buckets map[string]*bucket
}

func New(ratePerSecond, burst, maxIPs int) *Limiter {
	if ratePerSecond <= 0 {
		ratePerSecond = 50
	}

	if burst <= 0 {
		burst = 100
	}

	if maxIPs <= 0 {
		maxIPs = 20000
	}

	return &Limiter{
		rate:    float64(ratePerSecond),
		burst:   float64(burst),
		maxIPs:  maxIPs,
		buckets: make(map[string]*bucket),
	}
}

func (l *Limiter) Allow(addr net.Addr) bool {
	ip := remoteIP(addr)
	if ip == "" {
		return false
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]

	if !ok {
		if len(l.buckets) >= l.maxIPs {
			l.cleanupLocked(now)

			if len(l.buckets) >= l.maxIPs {
				return false
			}
		}

		l.buckets[ip] = &bucket{
			tokens: l.burst - 1,
			last:   now,
			seen:   now,
		}

		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate

		if b.tokens > l.burst {
			b.tokens = l.burst
		}
	}

	b.last = now
	b.seen = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(time.Now())
}

func (l *Limiter) cleanupLocked(now time.Time) {
	const maxIdle = 10 * time.Minute

	for ip, b := range l.buckets {
		if now.Sub(b.seen) > maxIdle {
			delete(l.buckets, ip)
		}
	}
}

func remoteIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	switch v := addr.(type) {
	case *net.UDPAddr:
		if v.IP == nil {
			return ""
		}
		return v.IP.String()

	case *net.TCPAddr:
		if v.IP == nil {
			return ""
		}
		return v.IP.String()
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}

	return host
}
