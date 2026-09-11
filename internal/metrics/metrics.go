package metrics

import "sync/atomic"

type Metrics struct {
	CacheHits     atomic.Uint64
	CacheMisses   atomic.Uint64
	UpstreamQuery atomic.Uint64
	UpstreamError atomic.Uint64
}

type Snapshot struct {
	CacheHits     uint64
	CacheMisses   uint64
	UpstreamQuery uint64
	UpstreamError uint64
}

func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		CacheHits:     m.CacheHits.Load(),
		CacheMisses:   m.CacheMisses.Load(),
		UpstreamQuery: m.UpstreamQuery.Load(),
		UpstreamError: m.UpstreamError.Load(),
	}
}
