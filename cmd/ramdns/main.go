package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/ramdns/ramdns/internal/cache"
	dnsinternal "github.com/ramdns/ramdns/internal/dns"
	"github.com/ramdns/ramdns/internal/filter"
	"github.com/ramdns/ramdns/internal/metrics"
)

const (
	listenAddr   = ":5353"
	upstreamAddr = "1.1.1.1:53"
	cacheEntries = 50000
)

type Resolver struct {
	client   *dns.Client
	upstream string

	cache   *cache.Cache
	filter  *filter.Filter
	metrics *metrics.Metrics
	flight  *dnsinternal.SingleFlight
}

func NewResolver() *Resolver {
	dnsFilter := filter.New()

	// Temporary local test rules.
	// These will later be replaced by
	// dynamically loaded blocklists.
	dnsFilter.Add("ads.example.com")
	dnsFilter.Add("tracker.example.com")
	dnsFilter.Add("telemetry.example.com")

	return &Resolver{
		client: &dns.Client{
			Net:     "udp",
			Timeout: 3 * time.Second,
		},

		upstream: upstreamAddr,

		cache:   cache.New(cacheEntries),
		filter:  dnsFilter,
		metrics: &metrics.Metrics{},
		flight:  dnsinternal.NewSingleFlight(),
	}
}

func (r *Resolver) Resolve(
	ctx context.Context,
	req *dns.Msg,
) *dns.Msg {

	key := cache.Key(req)

	if key == "" {
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeFormatError)
		return msg
	}

	// --------------------------------
	// FILTER
	// --------------------------------

	if len(req.Question) > 0 {
		qname := req.Question[0].Name

		if r.filter.IsBlocked(qname) {
			log.Printf(
				"filter BLOCK: %s",
				qname,
			)

			msg := new(dns.Msg)
			msg.SetRcode(req, dns.RcodeNameError)

			return msg
		}
	}

	// --------------------------------
	// CACHE HIT
	// --------------------------------

	if resp, ok := r.cache.Get(key); ok {
		r.metrics.CacheHits.Add(1)

		resp.Id = req.Id
		resp.Question = append(
			[]dns.Question(nil),
			req.Question...,
		)

		log.Printf(
			"cache HIT key=%s cache_size=%d",
			key,
			r.cache.Size(),
		)

		return resp
	}

	// --------------------------------
	// CACHE MISS
	// --------------------------------

	r.metrics.CacheMisses.Add(1)

	log.Printf(
		"cache MISS key=%s",
		key,
	)

	result := r.flight.Do(key, func() interface{} {

		// Re-check cache after entering
		// the single-flight section.

		if resp, ok := r.cache.Get(key); ok {
			return resp
		}

		r.metrics.UpstreamQuery.Add(1)

		resp, _, err := r.client.ExchangeContext(
			ctx,
			req,
			r.upstream,
		)

		if err != nil {
			r.metrics.UpstreamError.Add(1)

			log.Printf(
				"upstream error key=%s error=%v",
				key,
				err,
			)

			msg := new(dns.Msg)
			msg.SetRcode(
				req,
				dns.RcodeServerFailure,
			)

			return msg
		}

		r.cache.Set(key, resp)

		return resp
	})

	resp, ok := result.(*dns.Msg)

	if !ok || resp == nil {
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)
		return msg
	}

	resp = resp.Copy()

	resp.Id = req.Id
	resp.Question = append(
		[]dns.Question(nil),
		req.Question...,
	)

	return resp
}

func (r *Resolver) HandleDNS(
	w dns.ResponseWriter,
	req *dns.Msg,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	resp := r.Resolve(ctx, req)

	if err := w.WriteMsg(resp); err != nil {
		log.Printf(
			"response error: %v",
			err,
		)
	}
}

func main() {
	log.SetFlags(
		log.Ldate |
			log.Ltime |
			log.Lmicroseconds,
	)

	resolver := NewResolver()
	handler := dns.HandlerFunc(resolver.HandleDNS)

	udpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: handler,
	}

	tcpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "tcp",
		Handler: handler,
	}

	errChan := make(chan error, 2)

	go func() {
		log.Printf(
			"RAMDNS UDP listening on %s",
			listenAddr,
		)

		errChan <- udpServer.ListenAndServe()
	}()

	go func() {
		log.Printf(
			"RAMDNS TCP listening on %s",
			listenAddr,
		)

		errChan <- tcpServer.ListenAndServe()
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case sig := <-signalChan:
		log.Printf(
			"RAMDNS shutting down: %s",
			sig,
		)

	case err := <-errChan:
		log.Printf(
			"RAMDNS server error: %v",
			err,
		)
	}

	_ = udpServer.Shutdown()
	_ = tcpServer.Shutdown()

	log.Println("RAMDNS stopped")
}
