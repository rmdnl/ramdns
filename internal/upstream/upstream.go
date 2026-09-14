package upstream

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultTimeout = 3 * time.Second
	perServerTime  = 1500 * time.Millisecond
	dialTimeout    = 1500 * time.Millisecond
	idleTimeout    = 30 * time.Second

	healthInterval     = 10 * time.Second
	healthProbeTimeout = 1200 * time.Millisecond
	failuresBeforeDown = 3
	successesBeforeUp  = 2
)

type Server struct {
	Addr       string
	ServerName string
	Client     *dns.Client

	mu       sync.Mutex
	conn     *dns.Conn
	lastUsed time.Time

	queries   atomic.Uint64
	success   atomic.Uint64
	failures  atomic.Uint64
	latencyNS atomic.Int64

	healthMu         sync.Mutex
	consecutiveFails int
	consecutiveOK    int
	healthy          atomic.Bool
}

type Resolver struct {
	Servers []*Server

	stopHealth chan struct{}
	healthOnce sync.Once
	wg         sync.WaitGroup
}

func New(specs []string) (*Resolver, error) {
	if len(specs) == 0 {
		return nil, errors.New("no upstream servers configured")
	}

	servers := make([]*Server, 0, len(specs))

	for _, raw := range specs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		parts := strings.SplitN(raw, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf(
				"invalid upstream spec %q, expected addr|servername",
				raw,
			)
		}

		addr := strings.TrimSpace(parts[0])
		serverName := strings.TrimSpace(parts[1])

		if addr == "" || serverName == "" {
			return nil, fmt.Errorf("invalid upstream spec %q", raw)
		}

		server := &Server{
			Addr:       addr,
			ServerName: serverName,
			Client: &dns.Client{
				Net:     "tcp-tls",
				Timeout: defaultTimeout,
				TLSConfig: &tls.Config{
					ServerName:         serverName,
					MinVersion:         tls.VersionTLS13,
					InsecureSkipVerify: false,
				},
			},
		}

		server.healthy.Store(true)
		servers = append(servers, server)
	}

	if len(servers) == 0 {
		return nil, errors.New("no valid upstream servers configured")
	}

	r := &Resolver{
		Servers:    servers,
		stopHealth: make(chan struct{}),
	}

	r.StartHealthChecker()

	return r, nil
}

func (s *Server) closeConnLocked() {
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func (s *Server) closeConn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeConnLocked()
}

func (s *Server) exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil && time.Since(s.lastUsed) > idleTimeout {
		s.closeConnLocked()
	}

	if s.conn == nil {
		dialer := &net.Dialer{Timeout: dialTimeout}

		raw, err := dialer.DialContext(ctx, "tcp", s.Addr)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(raw, s.Client.TLSConfig.Clone())

		if deadline, ok := ctx.Deadline(); ok {
			_ = tlsConn.SetDeadline(deadline)
		}

		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = tlsConn.Close()
			return nil, err
		}

		s.conn = &dns.Conn{Conn: tlsConn}
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = s.conn.SetDeadline(deadline)
	} else {
		_ = s.conn.SetDeadline(time.Time{})
	}

	if err := s.conn.WriteMsg(req); err != nil {
		s.closeConnLocked()
		return nil, err
	}

	resp, err := s.conn.ReadMsg()
	if err != nil {
		s.closeConnLocked()
		return nil, err
	}

	s.lastUsed = time.Now()
	return resp, nil
}

func (s *Server) markSuccess() {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	wasHealthy := s.healthy.Load()

	s.consecutiveFails = 0
	s.consecutiveOK++

	if s.consecutiveOK >= successesBeforeUp {
		s.healthy.Store(true)
	}

	if !wasHealthy && s.healthy.Load() {
		log.Printf(
			"upstream recovered addr=%s server=%s",
			s.Addr,
			s.ServerName,
		)
	}
}

func (s *Server) markFailure() {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	wasHealthy := s.healthy.Load()

	s.consecutiveOK = 0
	s.consecutiveFails++

	if s.consecutiveFails >= failuresBeforeDown {
		s.healthy.Store(false)
	}

	if wasHealthy && !s.healthy.Load() {
		log.Printf(
			"upstream unhealthy addr=%s server=%s failures=%d",
			s.Addr,
			s.ServerName,
			s.consecutiveFails,
		)
	}
}

func (s *Server) isHealthy() bool {
	return s.healthy.Load()
}

func (s *Server) resetHealthForProbe() {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	s.consecutiveOK = 0
	s.consecutiveFails = 0
}

func (r *Resolver) healthyServers() []*Server {
	servers := make([]*Server, 0, len(r.Servers))

	for _, server := range r.Servers {
		if server != nil && server.isHealthy() {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		return r.Servers
	}

	return servers
}

func (r *Resolver) exchangeServer(
	ctx context.Context,
	server *Server,
	req *dns.Msg,
) (*dns.Msg, error) {
	start := time.Now()
	server.queries.Add(1)

	resp, err := server.exchange(ctx, req)

	latency := time.Since(start).Nanoseconds()
	server.latencyNS.Store(latency)

	if err != nil {
		server.failures.Add(1)
		server.markFailure()
		return nil, err
	}

	server.success.Add(1)
	server.markSuccess()
	return resp, nil
}

func (r *Resolver) Exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	if req == nil {
		return nil, errors.New("nil DNS request")
	}

	if len(r.Servers) == 0 {
		return nil, errors.New("no upstream servers configured")
	}

	servers := r.healthyServers()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		msg *dns.Msg
		err error
	}

	results := make(chan result, len(servers))
	var wg sync.WaitGroup

	for _, server := range servers {
		server := server

		wg.Add(1)
		go func() {
			defer wg.Done()

			timeout := perServerTime

			if deadline, ok := ctx.Deadline(); ok {
				remaining := time.Until(deadline)
				if remaining <= 0 {
					results <- result{err: context.DeadlineExceeded}
					return
				}

				if remaining < timeout {
					timeout = remaining
				}
			}

			serverCtx, serverCancel := context.WithTimeout(ctx, timeout)
			defer serverCancel()

			resp, err := r.exchangeServer(serverCtx, server, req)
			results <- result{
				msg: resp,
				err: err,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var lastErr error

	for res := range results {
		if res.err == nil && res.msg != nil {
			cancel()
			return res.msg, nil
		}

		if res.err != nil {
			lastErr = res.err
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if lastErr == nil {
		lastErr = errors.New("all upstream servers failed")
	}

	return nil, lastErr
}

func (r *Resolver) probeServer(server *Server) {
	req := new(dns.Msg)
	req.SetQuestion(".", dns.TypeNS)
	req.RecursionDesired = true

	ctx, cancel := context.WithTimeout(
		context.Background(),
		healthProbeTimeout,
	)
	defer cancel()

	resp, err := server.exchange(ctx, req)

	if err != nil || resp == nil {
		server.markFailure()
		return
	}

	switch resp.Rcode {
	case dns.RcodeSuccess, dns.RcodeNameError:
		server.markSuccess()

	default:
		server.markFailure()
	}
}

func (r *Resolver) healthCheck() {
	for _, server := range r.Servers {
		if server == nil {
			continue
		}

		r.probeServer(server)
	}
}

func (r *Resolver) StartHealthChecker() {
	if r == nil {
		return
	}

	r.healthOnce.Do(func() {
		r.wg.Add(1)

		go func() {
			defer r.wg.Done()

			ticker := time.NewTicker(healthInterval)
			defer ticker.Stop()

			log.Printf(
				"upstream health checker started interval=%s",
				healthInterval,
			)

			for {
				select {
				case <-ticker.C:
					r.healthCheck()

				case <-r.stopHealth:
					return
				}
			}
		}()
	})
}

func (r *Resolver) Close() {
	if r == nil {
		return
	}

	close(r.stopHealth)
	r.wg.Wait()

	for _, server := range r.Servers {
		if server != nil {
			server.closeConn()
		}
	}
}

type ServerHealth struct {
	Healthy   bool   `json:"healthy"`
	Queries   uint64 `json:"queries"`
	Successes uint64 `json:"successes"`
	Failures  uint64 `json:"failures"`
	LatencyNS int64  `json:"latency_ns"`
}

func (r *Resolver) Health() map[string]ServerHealth {
	if r == nil {
		return nil
	}

	result := make(map[string]ServerHealth, len(r.Servers))

	for _, server := range r.Servers {
		if server == nil {
			continue
		}

		result[server.Addr+"|"+server.ServerName] = ServerHealth{
			Healthy:   server.isHealthy(),
			Queries:   server.queries.Load(),
			Successes: server.success.Load(),
			Failures:  server.failures.Load(),
			LatencyNS: server.latencyNS.Load(),
		}
	}

	return result
}
