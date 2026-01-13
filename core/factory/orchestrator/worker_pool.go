package orchestrator

import (
	"sync"
	"sync/atomic"
	"time"
)

type WorkerPool struct {
	workers []*Worker
	mu      sync.RWMutex
	next    uint64 // Tracks the next worker index
	timeout time.Duration
	retries int
}

func NewWorkerPool(workers []*Worker, timeout time.Duration, retries int) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		mu:      sync.RWMutex{},
		next:    0,
		timeout: timeout,
		retries: retries,
	}
}

func (p *WorkerPool) GetNextWorker() *Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	n := len(p.workers)
	if n == 0 {
		return nil
	}

	// Atomic increment ensures thread-safety across goroutines.
	// We use the modulo operator (%) to wrap around the slice length.
	idx := atomic.AddUint64(&p.next, 1)
	return p.workers[(idx-1)%uint64(n)]
}

// SelectWorker returns the next worker in the pool (round robin load balancing)
func (p *WorkerPool) SelectWorker() *Worker {
	return p.GetNextWorker()
}
