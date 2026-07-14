package rate

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

func NewLimiter(rps float64, burst int) *Limiter {
	if rps <= 0 {
		rps = 100
	}
	if burst <= 0 {
		burst = 10
	}
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		r:        rate.Limit(rps),
		burst:    burst,
	}
}

func (l *Limiter) Wait(ctx context.Context, key string) error {
	lim := l.get(key)
	return lim.Wait(ctx)
}

func (l *Limiter) get(key string) *rate.Limiter {
	l.mu.RLock()
	lim, ok := l.limiters[key]
	l.mu.RUnlock()
	if ok {
		return lim
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[key]; ok {
		return lim
	}
	lim = rate.NewLimiter(l.r, l.burst)
	l.limiters[key] = lim
	return lim
}

type CircuitBreaker struct {
	mu              sync.Mutex
	failures        int
	state           string
	lastFailureTime time.Time
	threshold       int
	cooldown        time.Duration
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		state:     "closed",
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.cooldown {
			cb.state = "half-open"
			cb.failures = 0
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailureTime = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
