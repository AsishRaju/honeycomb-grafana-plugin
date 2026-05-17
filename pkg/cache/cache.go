// Package cache provides a thread-safe, TTL-based in-memory cache used to
// reduce Honeycomb API calls and protect against the 10 req/min rate limit
// on the Create Query Result endpoint.
//
// Architecture (see ADR-002):
//
//	L1: query shape key  → query_id        (TTL 1 h)
//	L2: execution key    → query_result_id  (TTL 30 min)
//	L3: query_result_id  → completed result (TTL 24 h)
//
// A singleflight group wraps the L2/L3 population path so that concurrent
// panels requesting the same data collapse into a single Honeycomb call.
package cache

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Generic TTL cache
// ---------------------------------------------------------------------------

type entry struct {
	value     interface{}
	expiresAt time.Time
}

// Cache is a simple thread-safe TTL store.
type Cache struct {
	mu    sync.RWMutex
	items map[string]entry

	// janitor goroutine runs every cleanupInterval to evict expired entries.
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

// New creates a Cache and starts a background janitor.
func New(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:           make(map[string]entry),
		cleanupInterval: cleanupInterval,
		stopCh:          make(chan struct{}),
	}
	go c.janitor()
	return c
}

// Set stores value under key with the given TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Get retrieves a value. Returns (value, true) if found and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Len returns the number of entries (including expired ones not yet cleaned up).
func (c *Cache) Len() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Stop shuts down the background janitor.
func (c *Cache) Stop() {
	close(c.stopCh)
}

func (c *Cache) janitor() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) deleteExpired() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Typed TTLs for each cache level
// ---------------------------------------------------------------------------

const (
	// TTLQueryID is the TTL for L1 (query shape → query_id).
	// Honeycomb queries are immutable once created; 1 h is conservative.
	TTLQueryID = 1 * time.Hour

	// TTLQueryResultID is the TTL for L2 (execution key → query_result_id).
	// A result_id may still be in a "pending" state; we retain it so we can
	// poll it rather than re-submit.
	TTLQueryResultID = 30 * time.Minute

	// TTLCompletedResult is the TTL for L3 (result_id → completed result).
	// Matches Honeycomb's Cache-Control: private, max-age=86400.
	TTLCompletedResult = 24 * time.Hour

	// TTLMetadata is the TTL for dataset and column metadata.
	TTLMetadata = 5 * time.Minute
)

// ---------------------------------------------------------------------------
// Singleflight for in-flight deduplication
// ---------------------------------------------------------------------------

// Group provides singleflight semantics: concurrent callers with the same key
// block until the first caller completes, then all receive the same result.
type Group struct {
	mu    sync.Mutex
	calls map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Do executes fn if no other goroutine is currently executing fn for the same
// key. All concurrent callers with the same key block until fn returns and
// then receive the same result.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*call)
	}
	if c, ok := g.calls[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true // shared (not the caller that executed fn)
	}
	c := &call{}
	c.wg.Add(1)
	g.calls[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return c.val, c.err, false // this caller executed fn
}
