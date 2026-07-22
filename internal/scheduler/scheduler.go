package scheduler

import (
	"context"
	"sync"
	"time"
)

type RefreshFunc func(ctx context.Context) error

type LogFunc func(level, msg string)

type Scheduler struct {
	interval      time.Duration
	refreshFn     RefreshFunc
	logFn         LogFunc
	cancel        context.CancelFunc
	mu            sync.Mutex
	running       bool
}

func New(interval int, fn RefreshFunc, logFn LogFunc) *Scheduler {
	return &Scheduler{
		interval:  time.Duration(interval) * time.Second,
		refreshFn: fn,
		logFn:     logFn,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if s.logFn != nil {
					s.logFn("info", "scheduler: auto refresh starting")
				}
				if err := s.refreshFn(ctx); err != nil {
					if s.logFn != nil {
						s.logFn("error", "scheduler: refresh error: "+err.Error())
					}
				}
			case <-ctx.Done():
				if s.logFn != nil {
					s.logFn("info", "scheduler: stopped")
				}
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}
