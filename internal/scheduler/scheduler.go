package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

type RefreshFunc func(ctx context.Context) error

type Scheduler struct {
	interval      time.Duration
	refreshFn     RefreshFunc
	cancel        context.CancelFunc
	mu            sync.Mutex
	running       bool
}

func New(interval int, fn RefreshFunc) *Scheduler {
	return &Scheduler{
		interval:  time.Duration(interval) * time.Second,
		refreshFn: fn,
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
				log.Println("scheduler: auto refresh starting")
				if err := s.refreshFn(ctx); err != nil {
					log.Printf("scheduler: refresh error: %v", err)
				}
			case <-ctx.Done():
				log.Println("scheduler: stopped")
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
