package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New(3600, func(ctx context.Context) error { return nil }, nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.interval != 3600*time.Second {
		t.Errorf("interval: got %v", s.interval)
	}
}

func TestStartAndStop(t *testing.T) {
	var count int32
	s := New(1, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx)

	time.Sleep(2500 * time.Millisecond)

	s.Stop()

	calls := atomic.LoadInt32(&count)
	if calls < 2 {
		t.Errorf("expected at least 2 calls in 2.5s with 1s interval, got %d", calls)
	}

	time.Sleep(1500 * time.Millisecond)
	afterStop := atomic.LoadInt32(&count)
	if afterStop != calls {
		t.Errorf("scheduler should not run after Stop: before=%d after=%d", calls, afterStop)
	}
}

func TestStart_Idempotent(t *testing.T) {
	var count int32
	s := New(100, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, nil)

	ctx := context.Background()
	s.Start(ctx)
	s.Start(ctx)
	s.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	s.Stop()

	calls := atomic.LoadInt32(&count)
	if calls > 3 {
		t.Errorf("expected few calls due to idempotent start and long interval, got %d", calls)
	}
}

func TestStop_NotStarted(t *testing.T) {
	s := New(3600, func(ctx context.Context) error { return nil }, nil)

	s.Stop()

	if s.running {
		t.Error("s.running should be false")
	}
}

func TestIndependentProviders(t *testing.T) {
	var count int32
	s := New(3600, func(ctx context.Context) error { return nil }, nil)
	s.Start(context.Background())
	defer s.Stop()

	got := map[string]bool{}
	s.SetIndependentProviders(func(ctx context.Context, name string) error {
		atomic.AddInt32(&count, 1)
		got[name] = true
		return nil
	}, map[string]time.Duration{
		"a": 100 * time.Millisecond,
		"b": 100 * time.Millisecond,
	})

	time.Sleep(350 * time.Millisecond)

	if atomic.LoadInt32(&count) < 2 {
		t.Errorf("expected independent refreshes, got %d", count)
	}
	if !got["a"] || !got["b"] {
		t.Errorf("expected refreshes for a and b, got %v", got)
	}

	// Replacing with an empty set stops the timers
	s.SetIndependentProviders(func(ctx context.Context, name string) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, map[string]time.Duration{})

	before := atomic.LoadInt32(&count)
	time.Sleep(300 * time.Millisecond)
	if atomic.LoadInt32(&count) != before {
		t.Errorf("independent timers should stop after empty set: before=%d after=%d", before, atomic.LoadInt32(&count))
	}
}
