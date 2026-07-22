package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New(3600, func(ctx context.Context) error { return nil })
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
	})

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
	})

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
	s := New(3600, func(ctx context.Context) error { return nil })

	s.Stop()

	if s.running {
		t.Error("s.running should be false")
	}
}
