# ADR-002: Multi-Level Caching Strategy

**Status:** Accepted  
**Date:** 2025-01-15

## Context

Honeycomb's Create Query Result endpoint is hard-capped at **10 requests per minute**. A typical Grafana dashboard with 10 panels, auto-refreshed every 30 seconds, would naturally generate 10 Create Query Result calls on each refresh — instantly hitting the cap. With 20 panels that's 20 calls, causing persistent 429s.

We need aggressive, correct caching that:
1. Reuses completed query results as long as they're fresh
2. Avoids creating duplicate in-flight requests
3. Shares the cache across all panels and users on the same Grafana instance
4. Respects Honeycomb's own caching semantics (`Cache-Control: private, max-age=86400`)

## Decision

Implement a **three-level in-process cache** in the Go backend, with singleflight deduplication as a fourth layer.

### Cache levels

| Level | Key | Value | TTL | Purpose |
|-------|-----|-------|-----|---------|
| L1 | `SHA256(dsUID + dataset + normalizedQueryShape)` | `query_id` | 1 h | Avoid re-creating identical query specs |
| L2 | `SHA256(query_id + disable_series + limit)` | `query_result_id` | 30 min | Avoid re-submitting a running result |
| L3 | `query_result_id` | completed `QueryResult` | 86 400 s (24 h) | Cache Honeycomb's completed result; matches their `Cache-Control: max-age=86400` |

**Singleflight (L0):** Before any cache check, a singleflight group keyed on the full request fingerprint ensures that N concurrent panels requesting identical data collapse into exactly one Honeycomb API call. All waiters receive the same response.

### Time range normalization for cache key stability

Grafana provides absolute `from`/`to` timestamps. We snap these to the same resolution that Honeycomb uses for result truncation:
- time_range ≤ 6 h: snap to nearest 60 s  
- time_range ≤ 48 h: snap to nearest 300 s  
- time_range ≤ 168 h: snap to nearest 1800 s

This means a dashboard that auto-refreshes every 30 s will reuse the same cache entry for the full cache TTL window.

### Query shape normalization

Before fingerprinting, we normalize the Honeycomb query:
- Sort `breakdowns`, `calculations`, `filters`, `orders` into a canonical order
- Omit zero-value / empty fields
- Produce deterministic JSON (sorted keys)

The normalized shape is the cache key component for L1; it ensures semantically identical queries with different field ordering share one cache entry.

## Consequences

**Positive:**
- In the steady state, a dashboard refresh hits only L3 (in-memory, microsecond latency)
- A new panel or new time range hits L1 (one Create Query call) + rate-limited Create Query Result, then polls Get Query Result
- Shared cache means user A's panel warms the cache for user B's identical panel

**Negative:**
- Cache is in-process; it resets on Grafana restart or plugin reload
- Cache is not distributed; multiple Grafana replicas each maintain their own cache. Acceptable for most deployments; document limitation.
- L3 TTL of 86 400 s is long. Users may see stale data for queries that haven't changed. Add a "force refresh" capability via a query option that bypasses L3.

## Alternatives Rejected

**Redis-backed distributed cache:** Adds operational complexity and a hard dependency. Overkill for a single-replica Grafana deployment, which is the common case. Can be added later.

**No caching, just rate limiting:** Rate limiter alone doesn't prevent stampedes. Under a burst of 20 panels, a 10/min limiter makes panels queue for 2 minutes before getting results. Unacceptable UX.

**Hash only on query_id for L2:** query_id doesn't encode `disable_series` or `limit`, so two panels with different display modes but identical queries would share a result_id incorrectly. The L2 key includes all result-affecting options.
