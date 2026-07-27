// Package ratelimit provides a token-bucket rate limiter for the
// Honeycomb Create Query Result endpoint.
//
// See ADR-003 for the original rationale. This Etsy fork raises the local
// client budget above Honeycomb's published 10/min hard cap (60/min, burst 30)
// so dense Grafana dashboards can submit more aggressively.
//
// The limiter is per-datasource-instance (one per API key), created in
// NewDatasource and held in the Datasource struct.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

const (
	// tokensPerMinute is the local Create Query Result budget.
	tokensPerMinute = 60
	// burstSize allows a short burst of back-to-back submissions at startup.
	burstSize = 30
	// maxWait is the longest we will block waiting for a token.
	maxWait = 30 * time.Second
)

// Limiter wraps golang.org/x/time/rate.Limiter with Honeycomb-specific settings.
type Limiter struct {
	rl *rate.Limiter
}

// New returns a Limiter configured for the Create Query Result endpoint.
func New() *Limiter {
	r := rate.Every(time.Minute / tokensPerMinute) // one token every 1 s
	return &Limiter{rl: rate.NewLimiter(r, burstSize)}
}

// Wait blocks until a token is available or ctx is cancelled.
// Returns an error if ctx expires before a token becomes available, or if
// the wait would exceed maxWait.
//
// Callers should call Wait immediately before invoking
// honeycomb.Client.CreateQueryResult.
func (l *Limiter) Wait(ctx context.Context) error {
	// Create a child context with a hard maxWait deadline so we do not block
	// indefinitely even if ctx has no deadline.
	waitCtx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	if err := l.rl.Wait(waitCtx); err != nil {
		if waitCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf(
				"timed out waiting for Honeycomb rate limit token after %s "+
					"(Create Query Result is capped at %d req/min on Honeycomb's side; "+
					"too many concurrent panels may be competing for capacity)",
				maxWait, tokensPerMinute)
		}
		return fmt.Errorf("rate limiter context cancelled: %w", err)
	}
	return nil
}

// Reserve returns an immediately-available token if possible without blocking.
// Returns false if no token is available right now (non-blocking probe).
func (l *Limiter) Reserve() bool {
	return l.rl.Allow()
}

// Tokens returns the current number of available tokens (may be fractional).
func (l *Limiter) Tokens() float64 {
	return l.rl.Tokens()
}
