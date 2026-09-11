package doh

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

const (
	Path        = "/dns-query"
	ContentType = "application/dns-message"
	MaxBodySize = 65535
)

type Resolver interface {
	Resolve(context.Context, *dns.Msg) *dns.Msg
}

type Handler struct {
	resolver Resolver
}

func NewHandler(resolver Resolver) *Handler {
	return &Handler{
		resolver: resolver,
	}
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != Path {
		http.NotFound(w, r)
		return
	}

	var (
		packet []byte
		err    error
	)

	switch r.Method {
	case http.MethodGet:
		packet, err = decodeGET(r)

	case http.MethodPost:
		packet, err = decodePOST(r)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if err != nil {
		http.Error(
			w,
			"bad DNS message",
			http.StatusBadRequest,
		)
		return
	}

	req := new(dns.Msg)

	if err := req.Unpack(packet); err != nil {
		http.Error(
			w,
			"invalid DNS message",
			http.StatusBadRequest,
		)
		return
	}

	if len(req.Question) == 0 {
		http.Error(
			w,
			"DNS question is required",
			http.StatusBadRequest,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		3*time.Second,
	)
	defer cancel()

	resp := h.resolver.Resolve(ctx, req)

	if resp == nil {
		http.Error(
			w,
			"DNS resolver returned no response",
			http.StatusBadGateway,
		)
		return
	}

	body, err := resp.Pack()
	if err != nil {
		http.Error(
			w,
			"failed to encode DNS response",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", ContentType)
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(body)
}

func decodeGET(r *http.Request) ([]byte, error) {
	value := r.URL.Query().Get("dns")

	if value == "" {
		return nil, errors.New("missing dns parameter")
	}

	if len(value) > 10000 {
		return nil, errors.New("dns parameter too large")
	}

	packet, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	if len(packet) == 0 || len(packet) > MaxBodySize {
		return nil, errors.New("DNS message size invalid")
	}

	return packet, nil
}

func decodePOST(r *http.Request) ([]byte, error) {
	if r.Header.Get("Content-Type") != ContentType {
		return nil, errors.New("invalid content type")
	}

	if r.ContentLength > MaxBodySize {
		return nil, errors.New("DNS message too large")
	}

	reader := io.LimitReader(
		r.Body,
		MaxBodySize+1,
	)

	packet, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if len(packet) == 0 || len(packet) > MaxBodySize {
		return nil, errors.New("DNS message size invalid")
	}

	return packet, nil
}
