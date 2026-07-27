package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/honeycombio/grafana-honeycomb-datasource/pkg/ratelimit"
)

func drainBurst(lim *ratelimit.Limiter) {
	for lim.Reserve() {
	}
}

func TestLimiter_AllowsBurstImmediately(t *testing.T) {
	lim := ratelimit.New()

	// A fresh limiter should allow an immediate burst without waiting.
	start := time.Now()
	for i := 0; i < 30; i++ {
		if err := lim.Wait(context.Background()); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	// Allow generous slack for slow CI environments.
	if elapsed > 500*time.Millisecond {
		t.Errorf("burst calls took too long: %s", elapsed)
	}
}

func TestLimiter_RespectsContextCancellation(t *testing.T) {
	lim := ratelimit.New()

	drainBurst(lim)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := lim.Wait(ctx)
	if err == nil {
		t.Error("expected error when context times out while waiting for token")
	}
}

func TestLimiter_ReturnsTokensAvailability(t *testing.T) {
	lim := ratelimit.New()

	tokens := lim.Tokens()
	if tokens <= 0 {
		t.Errorf("expected positive tokens on fresh limiter, got %f", tokens)
	}
}

func TestLimiter_Reserve_NonBlocking(t *testing.T) {
	lim := ratelimit.New()

	// First few calls should be non-blocking (burst).
	got := lim.Reserve()
	if !got {
		t.Error("expected Reserve() to succeed on fresh limiter")
	}
}

func TestLimiter_MaxWaitTimeout(t *testing.T) {
	lim := ratelimit.New()

	drainBurst(lim)

	// With the burst drained, Wait should timeout rather than blocking
	// indefinitely. We use a very short context to avoid a slow test.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := lim.Wait(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected timeout error")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Wait should have failed quickly, took %s", elapsed)
	}
}
