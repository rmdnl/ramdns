package upstream

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultTimeout = 3 * time.Second
	perServerTime  = 1500 * time.Millisecond
)

type Server struct {
	Addr       string
	ServerName string
	Client     *dns.Client
}

type Resolver struct {
	Servers []*Server
}

func New(specs []string) (*Resolver, error) {
	if len(specs) == 0 {
		return nil, errors.New("no upstream servers configured")
	}

	servers := make([]*Server, 0, len(specs))

	for _, spec := range specs {
		spec = strings.TrimSpace(spec)

		if spec == "" {
			continue
		}

		parts := strings.SplitN(spec, "|", 2)

		if len(parts) != 2 {
			return nil, fmt.Errorf(
				"invalid upstream specification: %q",
				spec,
			)
		}

		addr := strings.TrimSpace(parts[0])
		serverName := strings.TrimSpace(parts[1])

		if addr == "" || serverName == "" {
			return nil, fmt.Errorf(
				"invalid upstream specification: %q",
				spec,
			)
		}

		if _, _, err := net.SplitHostPort(addr); err != nil {
			return nil, fmt.Errorf(
				"invalid upstream address %q: %w",
				addr,
				err,
			)
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

	return &Resolver{
		Servers: servers,
	}, nil
}

func (r *Resolver) Exchange(
	ctx context.Context,
	req *dns.Msg,
) (*dns.Msg, error) {
	if req == nil {
		return nil, errors.New("nil DNS request")
	}

	var lastErr error

	for _, server := range r.Servers {
		if server == nil || server.Client == nil {
			continue
		}

		remaining := time.Until(deadline(ctx))

		if remaining <= 0 {
			return nil, ctx.Err()
		}

		timeout := perServerTime
		if remaining < timeout {
			timeout = remaining
		}

		serverCtx, cancel := context.WithTimeout(ctx, timeout)

		resp, _, err := server.Client.ExchangeContext(
			serverCtx,
			req,
			server.Addr,
		)

		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	if lastErr == nil {
		lastErr = errors.New("all upstreams failed")
	}

	return nil, lastErr
}

func deadline(ctx context.Context) time.Time {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline
	}

	return time.Now().Add(defaultTimeout)
}
