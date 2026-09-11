package adlist

import (
	"context"
	"log"
	"strings"
	"time"
)

type Worker struct {
	manager  *Manager
	urls     []string
	interval time.Duration
}

func NewWorker(
	manager *Manager,
	rawURLs string,
	interval time.Duration,
) *Worker {
	return &Worker{
		manager:  manager,
		urls:     normalizeURLs(strings.Split(rawURLs, ",")),
		interval: interval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if len(w.urls) == 0 {
		log.Printf("adlist disabled: no URLs configured")
		return
	}

	// Load immediately at startup.
	w.load(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.load(ctx)
		}
	}
}

func (w *Worker) load(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()

	start := time.Now()

	err := w.manager.LoadURLs(ctx, w.urls)
	if err != nil {
		if err == ErrNotModified {
			log.Printf(
				"adlist unchanged sources=%d rules=%d duration=%s",
				len(w.urls),
				w.manager.RuleCount(),
				time.Since(start).Round(time.Millisecond),
			)
			return
		}

		log.Printf(
			"adlist update failed sources=%d rules=%d error=%v",
			len(w.urls),
			w.manager.RuleCount(),
			err,
		)
		return
	}

	log.Printf(
		"adlist updated sources=%d rules=%d duration=%s",
		len(w.urls),
		w.manager.RuleCount(),
		time.Since(start).Round(time.Millisecond),
	)
}
