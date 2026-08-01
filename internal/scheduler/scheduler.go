package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RefreshFunc func(ctx context.Context) error

type LogFunc func(level, msg string)

// IndependentRefreshFunc refreshes a single provider by name.
type IndependentRefreshFunc func(ctx context.Context, name string) error

type Scheduler struct {
	interval      time.Duration
	refreshFn     RefreshFunc
	independentFn IndependentRefreshFunc
	logFn         LogFunc
	cancel        context.CancelFunc
	provCancel    map[string]context.CancelFunc
	provInterval  map[string]time.Duration
	mu            sync.Mutex
	running       bool
}

func New(interval int, fn RefreshFunc, logFn LogFunc) *Scheduler {
	return &Scheduler{
		interval:     time.Duration(interval) * time.Second,
		refreshFn:    fn,
		logFn:        logFn,
		provCancel:   make(map[string]context.CancelFunc),
		provInterval: make(map[string]time.Duration),
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
	intervals := make(map[string]time.Duration, len(s.provInterval))
	for name, d := range s.provInterval {
		intervals[name] = d
	}
	s.mu.Unlock()
	go s.run(ctx)
	s.startIndependent(intervals)
}

func (s *Scheduler) run(ctx context.Context) {
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
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	for _, cancel := range s.provCancel {
		cancel()
	}
	s.provCancel = make(map[string]context.CancelFunc)
	s.running = false
}

func (s *Scheduler) UpdateInterval(ctx context.Context, interval int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = time.Duration(interval) * time.Second
	if s.running {
		if s.cancel != nil {
			s.cancel()
		}
		ctx, s.cancel = context.WithCancel(ctx)
		go s.run(ctx)
	}
}

// SetIndependentProviders replaces the set of per-provider timers.
// Each provider in the map gets its own dedicated timer with its own interval,
// refreshed via fn(name). Providers absent from the map are unscheduled.
func (s *Scheduler) SetIndependentProviders(fn IndependentRefreshFunc, providers map[string]time.Duration) {
	s.mu.Lock()
	s.independentFn = fn
	s.provInterval = make(map[string]time.Duration, len(providers))
	for name, d := range providers {
		s.provInterval[name] = d
	}
	running := s.running
	s.mu.Unlock()
	if running {
		s.startIndependent(providers)
	}
}

func (s *Scheduler) startIndependent(providers map[string]time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, cancel := range s.provCancel {
		_ = name
		cancel()
	}
	s.provCancel = make(map[string]context.CancelFunc)

	if s.independentFn == nil || !s.running {
		return
	}
	for name, interval := range providers {
		ctx, cancel := context.WithCancel(context.Background())
		s.provCancel[name] = cancel
		go s.runProvider(ctx, name, interval)
	}
}

func (s *Scheduler) runProvider(ctx context.Context, name string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.independentFn(ctx, name); err != nil {
				if s.logFn != nil {
					s.logFn("error", fmt.Sprintf("scheduler: independent refresh %s: %v", name, err))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
