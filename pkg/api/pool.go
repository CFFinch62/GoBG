package api

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages concurrent request processing with configurable limits.
// It provides separate pools for fast (evaluate) and slow (rollout) operations.
type WorkerPool struct {
	fastSem    chan struct{} // Semaphore for fast operations (evaluate, move, cube)
	slowSem    chan struct{} // Semaphore for slow operations (rollout)
	queuedFast int64         // Number of queued fast requests
	queuedSlow int64         // Number of queued slow requests
	activeFast int64         // Number of active fast requests
	activeSlow int64         // Number of active slow requests
	totalFast  int64         // Total fast requests processed
	totalSlow  int64         // Total slow requests processed
	mu         sync.RWMutex
}

// PoolConfig configures the worker pool.
type PoolConfig struct {
	MaxFastWorkers int // Max concurrent fast operations (default: 100)
	MaxSlowWorkers int // Max concurrent slow operations (default: 4)
}

// DefaultPoolConfig returns a PoolConfig with sensible defaults.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxFastWorkers: 100,
		MaxSlowWorkers: 4,
	}
}

// NewWorkerPool creates a new worker pool with the given configuration.
func NewWorkerPool(config PoolConfig) *WorkerPool {
	if config.MaxFastWorkers <= 0 {
		config.MaxFastWorkers = 100
	}
	if config.MaxSlowWorkers <= 0 {
		config.MaxSlowWorkers = 4
	}

	return &WorkerPool{
		fastSem: make(chan struct{}, config.MaxFastWorkers),
		slowSem: make(chan struct{}, config.MaxSlowWorkers),
	}
}

// AcquireFast acquires a slot for a fast operation.
// Returns an error if the context is cancelled while waiting.
func (p *WorkerPool) AcquireFast(ctx context.Context) error {
	atomic.AddInt64(&p.queuedFast, 1)
	defer atomic.AddInt64(&p.queuedFast, -1)

	select {
	case p.fastSem <- struct{}{}:
		atomic.AddInt64(&p.activeFast, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseFast releases a fast operation slot.
func (p *WorkerPool) ReleaseFast() {
	atomic.AddInt64(&p.activeFast, -1)
	atomic.AddInt64(&p.totalFast, 1)
	<-p.fastSem
}

// AcquireSlow acquires a slot for a slow operation.
// Returns an error if the context is cancelled while waiting.
func (p *WorkerPool) AcquireSlow(ctx context.Context) error {
	atomic.AddInt64(&p.queuedSlow, 1)
	defer atomic.AddInt64(&p.queuedSlow, -1)

	select {
	case p.slowSem <- struct{}{}:
		atomic.AddInt64(&p.activeSlow, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseSlow releases a slow operation slot.
func (p *WorkerPool) ReleaseSlow() {
	atomic.AddInt64(&p.activeSlow, -1)
	atomic.AddInt64(&p.totalSlow, 1)
	<-p.slowSem
}

// Stats returns current pool statistics.
type PoolStats struct {
	ActiveFast  int64 `json:"active_fast"`
	ActiveSlow  int64 `json:"active_slow"`
	QueuedFast  int64 `json:"queued_fast"`
	QueuedSlow  int64 `json:"queued_slow"`
	TotalFast   int64 `json:"total_fast"`
	TotalSlow   int64 `json:"total_slow"`
	MaxFast     int   `json:"max_fast"`
	MaxSlow     int   `json:"max_slow"`
}

// Stats returns current pool statistics.
func (p *WorkerPool) Stats() PoolStats {
	return PoolStats{
		ActiveFast: atomic.LoadInt64(&p.activeFast),
		ActiveSlow: atomic.LoadInt64(&p.activeSlow),
		QueuedFast: atomic.LoadInt64(&p.queuedFast),
		QueuedSlow: atomic.LoadInt64(&p.queuedSlow),
		TotalFast:  atomic.LoadInt64(&p.totalFast),
		TotalSlow:  atomic.LoadInt64(&p.totalSlow),
		MaxFast:    cap(p.fastSem),
		MaxSlow:    cap(p.slowSem),
	}
}

// TryAcquireFast tries to acquire a fast slot without blocking.
// Returns true if acquired, false if pool is full.
func (p *WorkerPool) TryAcquireFast() bool {
	select {
	case p.fastSem <- struct{}{}:
		atomic.AddInt64(&p.activeFast, 1)
		return true
	default:
		return false
	}
}

// TryAcquireSlow tries to acquire a slow slot without blocking.
// Returns true if acquired, false if pool is full.
func (p *WorkerPool) TryAcquireSlow() bool {
	select {
	case p.slowSem <- struct{}{}:
		atomic.AddInt64(&p.activeSlow, 1)
		return true
	default:
		return false
	}
}

// AcquireSlowWithTimeout tries to acquire a slow slot with a timeout.
func (p *WorkerPool) AcquireSlowWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.AcquireSlow(ctx)
}

