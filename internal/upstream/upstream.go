package upstream

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultTimeout = 3 * time.Second
	perServerTime  = 1500 * time.Millisecond
	dialTimeout    = 1500 * time.Millisecond
	idleTimeout    = 30 * time.Second
)

type Server struct {
	Addr       string
	ServerName string
	Client     *dns.Client

	mu       sync.Mutex
	conn     *dns.Conn
	lastUsed time.Time
}

type Resolver struct {
	Servers []*Server
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
			return nil, fmt.Errorf("invalid upstream spec %q, expected addr|servername", raw)
		}

		addr := strings.TrimSpace(parts[0])
		serverName := strings.TrimSpace(parts[1])

		if addr == "" || serverName == "" {
			return nil, fmt.Errorf("invalid upstream spec %q", raw)
		}

		servers = append(servers, &Server{
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
		})
	}

	if len(servers) == 0 {
		return nil, errors.New("no valid upstream servers configured")
	}

	return &Resolver{Servers: servers}, nil
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

func (r *Resolver) Exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	if req == nil {
		return nil, errors.New("nil DNS request")
	}

	if len(r.Servers) == 0 {
		return nil, errors.New("no upstream servers configured")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		msg *dns.Msg
		err error
	}

	results := make(chan result, len(r.Servers))
	var wg sync.WaitGroup

	for _, server := range r.Servers {
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

			resp, err := server.exchange(serverCtx, req)
			results <- result{msg: resp, err: err}
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

func (r *Resolver) Close() {
	if r == nil {
		return
	}

	for _, server := range r.Servers {
		if server != nil {
			server.closeConn()
		}
	}
}
