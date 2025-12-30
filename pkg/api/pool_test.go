package api

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWorkerPoolBasic(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 2,
		MaxSlowWorkers: 1,
	})

	// Test fast worker acquisition
	ctx := context.Background()
	if err := pool.AcquireFast(ctx); err != nil {
		t.Fatalf("Failed to acquire fast worker: %v", err)
	}

	stats := pool.Stats()
	if stats.ActiveFast != 1 {
		t.Errorf("Expected 1 active fast worker, got %d", stats.ActiveFast)
	}

	pool.ReleaseFast()
	stats = pool.Stats()
	if stats.ActiveFast != 0 {
		t.Errorf("Expected 0 active fast workers after release, got %d", stats.ActiveFast)
	}
	if stats.TotalFast != 1 {
		t.Errorf("Expected 1 total fast request, got %d", stats.TotalFast)
	}
}

func TestWorkerPoolSlowOperations(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 10,
		MaxSlowWorkers: 2,
	})

	ctx := context.Background()

	// Acquire both slow slots
	if err := pool.AcquireSlow(ctx); err != nil {
		t.Fatalf("Failed to acquire slow worker 1: %v", err)
	}
	if err := pool.AcquireSlow(ctx); err != nil {
		t.Fatalf("Failed to acquire slow worker 2: %v", err)
	}

	stats := pool.Stats()
	if stats.ActiveSlow != 2 {
		t.Errorf("Expected 2 active slow workers, got %d", stats.ActiveSlow)
	}

	// Try to acquire a third - should block
	if pool.TryAcquireSlow() {
		t.Error("Should not be able to acquire third slow worker")
	}

	pool.ReleaseSlow()
	pool.ReleaseSlow()

	stats = pool.Stats()
	if stats.TotalSlow != 2 {
		t.Errorf("Expected 2 total slow requests, got %d", stats.TotalSlow)
	}
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 1,
		MaxSlowWorkers: 1,
	})

	// Fill the pool
	ctx := context.Background()
	if err := pool.AcquireFast(ctx); err != nil {
		t.Fatalf("Failed to acquire fast worker: %v", err)
	}

	// Try to acquire with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := pool.AcquireFast(cancelCtx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	pool.ReleaseFast()
}

func TestWorkerPoolConcurrency(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 5,
		MaxSlowWorkers: 2,
	})

	var wg sync.WaitGroup
	ctx := context.Background()

	// Launch 10 fast workers - only 5 should run concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := pool.AcquireFast(ctx); err != nil {
				t.Errorf("Failed to acquire fast worker: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
			pool.ReleaseFast()
		}()
	}

	wg.Wait()

	stats := pool.Stats()
	if stats.TotalFast != 10 {
		t.Errorf("Expected 10 total fast requests, got %d", stats.TotalFast)
	}
}

func TestWorkerPoolTimeout(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 1,
		MaxSlowWorkers: 1,
	})

	// Fill the slow pool
	ctx := context.Background()
	if err := pool.AcquireSlow(ctx); err != nil {
		t.Fatalf("Failed to acquire slow worker: %v", err)
	}

	// Try to acquire with timeout
	err := pool.AcquireSlowWithTimeout(10 * time.Millisecond)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	pool.ReleaseSlow()
}

func TestWorkerPoolStats(t *testing.T) {
	pool := NewWorkerPool(PoolConfig{
		MaxFastWorkers: 10,
		MaxSlowWorkers: 4,
	})

	stats := pool.Stats()
	if stats.MaxFast != 10 {
		t.Errorf("Expected MaxFast=10, got %d", stats.MaxFast)
	}
	if stats.MaxSlow != 4 {
		t.Errorf("Expected MaxSlow=4, got %d", stats.MaxSlow)
	}
}

