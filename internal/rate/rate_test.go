package rate

import (
	"context"
	"testing"
	"time"
)

func TestLimiterWait(t *testing.T) {
	l := NewLimiter(1000, 10)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := l.Wait(ctx, "openai"); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Hour)
	if !cb.Allow() {
		t.Fatal("new circuit breaker should allow")
	}
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.Allow() {
		t.Fatal("circuit breaker should be open after 2 failures")
	}
	if cb.State() != "open" {
		t.Errorf("expected state open, got %s", cb.State())
	}
}

func TestCircuitBreakerSuccessCloses(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Hour)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	if !cb.Allow() {
		t.Fatal("circuit breaker should close after success")
	}
	if cb.State() != "closed" {
		t.Errorf("expected state closed, got %s", cb.State())
	}
}
