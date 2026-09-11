package adlist

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	manager  *Manager
	url      string
	interval time.Duration
}

func NewWorker(
	manager *Manager,
	url string,
	interval time.Duration,
) *Worker {
	return &Worker{
		manager:  manager,
		url:      url,
		interval: interval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w.manager == nil || w.url == "" {
		return
	}

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
	ctx, cancel := context.WithTimeout(
		parent,
		20*time.Second,
	)
	defer cancel()

	if err := w.manager.LoadURL(ctx, w.url); err != nil {
		log.Printf(
			"adlist update failed url=%s error=%v",
			w.url,
			err,
		)

		return
	}

	log.Printf(
		"adlist update successful url=%s rules=%d",
		w.url,
		w.manager.RuleCount(),
	)
}
