package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/ramdns/ramdns/internal/adlist"
	"github.com/ramdns/ramdns/internal/cache"
	dnsinternal "github.com/ramdns/ramdns/internal/dns"
	"github.com/ramdns/ramdns/internal/doh"
	"github.com/ramdns/ramdns/internal/filter"
	"github.com/ramdns/ramdns/internal/metrics"
	"github.com/ramdns/ramdns/internal/ratelimit"
	"github.com/ramdns/ramdns/internal/upstream"
)

const (
	listenAddr   = ":53"
	dotListen    = ":853"
	dohListen    = ":443"
	cacheEntries = 50000

	tlsCertFile = "/etc/ramdns/tls/fullchain.pem"
	tlsKeyFile  = "/etc/ramdns/tls/privkey.pem"
)

type Resolver struct {
	upstream *upstream.Resolver
	cache    *cache.Cache
	filter   *filter.Filter
	metrics  *metrics.Metrics
	flight   *dnsinternal.SingleFlight
}

func NewResolver() *Resolver {
	upstreamSpecs := strings.TrimSpace(os.Getenv("RAMDNS_UPSTREAMS"))

	if upstreamSpecs == "" {
		upstreamSpecs = "1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
	}

	specs := strings.Split(upstreamSpecs, ",")
	upstreamResolver, err := upstream.New(specs)
	if err != nil {
		log.Fatalf("failed to initialize upstream resolver: %v", err)
	}

	return &Resolver{
		upstream: upstreamResolver,
		cache:    cache.New(cacheEntries),
		filter:   filter.New(),
		metrics:  &metrics.Metrics{},
		flight:   dnsinternal.NewSingleFlight(),
	}
}

func (r *Resolver) Resolve(ctx context.Context, req *dns.Msg) *dns.Msg {
	if req == nil {
		return nil
	}

	key := cache.Key(req)
	if key == "" {
		resp := new(dns.Msg)
		resp.SetRcode(req, dns.RcodeFormatError)
		return resp
	}

	if r.filter.IsBlocked(req.Question[0].Name) {
		log.Printf("blocked query=%s", req.Question[0].Name)

		resp := new(dns.Msg)
		resp.SetRcode(req, dns.RcodeNameError)
		return resp
	}

	if cached, ok := r.cache.Get(key); ok {
		r.metrics.CacheHits.Add(1)

		cached.Id = req.Id
		cached.Question = append([]dns.Question(nil), req.Question...)

		return cached
	}

	r.metrics.CacheMisses.Add(1)

	result := r.flight.Do(key, func() interface{} {
		if cached, ok := r.cache.Get(key); ok {
			return cached
		}

		r.metrics.UpstreamQuery.Add(1)

		resp, err := r.upstream.Exchange(ctx, req)
		if err != nil {
			r.metrics.UpstreamError.Add(1)

			log.Printf(
				"upstream error qname=%s qtype=%d err=%v",
				req.Question[0].Name,
				req.Question[0].Qtype,
				err,
			)

			msg := new(dns.Msg)
			msg.SetRcode(req, dns.RcodeServerFailure)
			return msg
		}

		r.cache.Set(key, resp)

		return resp
	})

	if result == nil {
		resp := new(dns.Msg)
		resp.SetRcode(req, dns.RcodeServerFailure)
		return resp
	}

	resp, ok := result.(*dns.Msg)
	if !ok || resp == nil {
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)
		return msg
	}

	resp.Id = req.Id
	resp.Question = append([]dns.Question(nil), req.Question...)

	return resp
}

func (r *Resolver) HandleDNS(w dns.ResponseWriter, req *dns.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp := r.Resolve(ctx, req)

	if resp == nil {
		resp = new(dns.Msg)
		resp.SetRcode(req, dns.RcodeServerFailure)
	}

	if err := w.WriteMsg(resp); err != nil {
		log.Printf("dns response write error: %v", err)
	}
}

type RateLimitedDNSHandler struct {
	resolver *Resolver
	limiter  *ratelimit.Limiter
}

func NewRateLimitedDNSHandler(
	resolver *Resolver,
	limiter *ratelimit.Limiter,
) dns.Handler {
	return dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if req == nil {
			return
		}

		if !limiter.Allow(w.RemoteAddr()) {
			msg := new(dns.Msg)
			msg.SetRcode(req, dns.RcodeRefused)

			if err := w.WriteMsg(msg); err != nil {
				log.Printf("rate limit response error: %v", err)
			}

			return
		}

		resolver.HandleDNS(w, req)
	})
}

func loadTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
	if err != nil {
		log.Fatalf("failed to load TLS certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
}

func main() {
	resolver := NewResolver()

	/*
		Adlist worker.
	*/
	adlistURL := strings.TrimSpace(os.Getenv("RAMDNS_ADLIST_URL"))

	if adlistURL != "" {
		manager := adlist.NewManager(resolver.filter)

		worker := adlist.NewWorker(
			manager,
			adlistURL,
			6*time.Hour,
		)

		go worker.Run(context.Background())
	}

	/*
		Public DNS rate limiter.

		50 requests/sec per IP
		100 request burst
		20,000 tracked IPs maximum
	*/
	publicDNSLimiter := ratelimit.New(50, 100, 20000)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			publicDNSLimiter.Cleanup()
		}
	}()

	publicDNSHandler := NewRateLimitedDNSHandler(
		resolver,
		publicDNSLimiter,
	)

	/*
		Normal DNS handler for encrypted transports.
	*/
	secureDNSHandler := dns.HandlerFunc(resolver.HandleDNS)

	/*
		UDP/TCP DNS :53
	*/
	udpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: publicDNSHandler,
	}

	tcpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "tcp",
		Handler: publicDNSHandler,
	}

	/*
		DoT :853
	*/
	tlsConfig := loadTLSConfig()

	dotServer := &dns.Server{
		Addr:      dotListen,
		Net:       "tcp-tls",
		Handler:   secureDNSHandler,
		TLSConfig: tlsConfig,
	}

	/*
		DoH :443
	*/
	dohServer := &http.Server{
		Addr:    dohListen,
		Handler: doh.NewHandler(resolver),

		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 * 1024,
		TLSConfig:         tlsConfig,
	}

	/*
		Start DNS UDP.
	*/
	go func() {
		log.Printf("starting DNS UDP server on %s", listenAddr)

		if err := udpServer.ListenAndServe(); err != nil {
			log.Fatalf("DNS UDP server failed: %v", err)
		}
	}()

	/*
		Start DNS TCP.
	*/
	go func() {
		log.Printf("starting DNS TCP server on %s", listenAddr)

		if err := tcpServer.ListenAndServe(); err != nil {
			log.Fatalf("DNS TCP server failed: %v", err)
		}
	}()

	/*
		Start DoT.
	*/
	go func() {
		log.Printf("starting DoT server on %s", dotListen)

		if err := dotServer.ListenAndServe(); err != nil {
			log.Fatalf("DoT server failed: %v", err)
		}
	}()

	/*
		Start DoH.
	*/
	go func() {
		log.Printf("starting DoH server on %s", dohListen)

		if err := dohServer.ListenAndServeTLS("", ""); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("DoH server failed: %v", err)
		}
	}()

	log.Printf("RAMDNS started")
	log.Printf("DNS UDP/TCP: %s", listenAddr)
	log.Printf("DoT: %s", dotListen)
	log.Printf("DoH: %s", dohListen)

	/*
		Graceful shutdown.
	*/
	sigCh := make(chan os.Signal, 1)
	signal.Notify(
		sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-sigCh

	log.Printf("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := dohServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("DoH shutdown error: %v", err)
	}

	if err := dotServer.Shutdown(); err != nil {
		log.Printf("DoT shutdown error: %v", err)
	}

	if err := udpServer.Shutdown(); err != nil {
		log.Printf("DNS UDP shutdown error: %v", err)
	}

	if err := tcpServer.Shutdown(); err != nil {
		log.Printf("DNS TCP shutdown error: %v", err)
	}

	log.Printf("RAMDNS stopped")
}
