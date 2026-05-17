# How Caching Works

## The problem

Honeycomb limits the **Create Query Result** endpoint to **10 requests per minute** per API key. This is a hard limit with no exceptions.

A typical Grafana dashboard with 15 panels, each using a different query, auto-refreshing every 30 seconds would generate 15 requests per refresh — exceeding the limit on every cycle.

## The solution: three-level cache

The plugin maintains three in-process cache layers in the Go backend:

```
Request → singleflight → L3? → L2? → L1? → Honeycomb API
                                              ↓
                                        Token bucket limiter
                                        (8 tokens/60 s)
```

### L1: Query ID cache (TTL: 1 hour)

**Key:** `SHA256(datasource_uid + dataset + normalized_query_shape)`  
**Value:** Honeycomb `query_id`

The Honeycomb Query API (POST /1/queries) creates an immutable query specification and returns a stable ID. The plugin caches this ID so the same logical query (same calculations, filters, breakdowns) reuses the same Honeycomb query_id regardless of time range changes.

**Time range normalization:** The L1 key does NOT include the time range, so a panel scrolling through time still reuses the same query_id.

**Cache miss:** One call to POST /1/queries. This endpoint is NOT subject to the 10/min limit.

### L2: Query Result ID cache (TTL: 30 minutes)

**Key:** `SHA256(datasource_uid + dataset + normalized_query + snapped_from + snapped_to + disable_series + limit)`  
**Value:** Honeycomb `query_result_id`

Once a query has been submitted for execution, its result ID is cached so that repeated identical requests skip the rate-limited Create Query Result step.

**Time snapping:** The time range boundaries are snapped to Honeycomb's own truncation intervals:
- ≤6 h range: snap to nearest 60 s
- ≤48 h range: snap to nearest 300 s
- ≤7 d range: snap to nearest 1800 s

This means a panel refreshing every 30 seconds reuses the same L2 entry for the full 60-second bucket.

**Cache miss:** One call to POST /1/query_results. This IS the rate-limited endpoint. A token must be acquired from the token bucket before this call.

### L3: Completed result cache (TTL: 24 hours)

**Key:** `"result:" + execKey` (same as L2 key prefix)  
**Value:** Full `QueryResultResponse` from Honeycomb

Completed query results are immutable. Honeycomb returns `Cache-Control: private, max-age=86400` on completed results. The plugin honours this by caching for 24 hours.

**Cache hit:** No Honeycomb API calls at all. Results are served from memory in microseconds.

### Singleflight deduplication (L0)

Before any cache check, a singleflight group collapses concurrent requests for the same key into a single execution. If 10 panels all load simultaneously with the same query, only one execution path reaches the Honeycomb API — the other 9 wait and receive the same result.

This is critical for dashboard loads where all panels render in parallel.

---

## Cache lifecycle example

### Cold start (first dashboard load)

```
Panel A: execKey = abc123
  → L3 miss (no completed result)
  → L1 miss (no query_id)
  → POST /1/queries → query_id = qid-1 (cached in L1)
  → L2 miss (no result_id)
  → wait for token bucket...
  → POST /1/query_results → result_id = rid-1 (cached in L2)
  → poll GET /1/query_results/rid-1 until complete
  → cache result in L3 (24h TTL)
  → return frames

Panel B: same query as A (same execKey = abc123)
  → singleflight: shares Panel A's execution (blocked, then receives same result)
  → no additional Honeycomb calls
```

### Dashboard refresh (30 seconds later)

```
Same panel query, same time range bucket:
  → L3 hit! Return in microseconds.
  → 0 Honeycomb API calls
```

### Time range change

```
User moves time range forward by 10 minutes (still within 60s snap bucket):
  → L3 hit (snapped key unchanged)
  → 0 Honeycomb API calls

User moves time range to a new day (different snap bucket):
  → L3 miss
  → L1 hit (query_id still cached for 1h)
  → L2 miss (new time window)
  → Token bucket → POST /1/query_results → ...
  → 1 rate-limited call, then polling
```

---

## Rate limiter details

The token bucket is configured at **8 tokens per 60 seconds** (not 10) to leave headroom for:
- Other clients sharing the same API key (CLI, CI jobs, etc.)
- Clock skew between the plugin and Honeycomb's window boundaries
- Burst of a few back-to-back panel loads

**Burst size:** 3 tokens. A dashboard opening cold can immediately submit 3 queries in parallel, then waits ~7.5 s per additional query.

**On 429:** Honeycomb's Query Data API does not include a `Retry-After` header on 429 responses (per their docs). The plugin uses exponential backoff: 10 s → 20 s → 40 s (with 25% jitter), max 3 retries, then returns an error to Grafana.

**On high-429 environments:** If you share an API key with many other clients and see persistent 429s, either:
1. Create a dedicated Grafana API key (recommended)
2. Increase the panel time range to benefit from L3 caching (longer TTL coverage)
3. Reduce dashboard refresh frequency

---

## Cache observability

The plugin logs cache hits at **DEBUG** level and misses/API calls at **DEBUG/INFO** level. To see cache activity, set Grafana's log level to debug:

```ini
[log]
level = debug
filters = honeycombio-honeycomb-datasource:debug
```

Example log output:
```
DEBUG L3 cache hit (completed result)  dataset=production key=abc123...
DEBUG L1 cache hit (query_id)          dataset=production shape_key=def456...
DEBUG Created query, cached in L1      dataset=production query_id=qid-abc
WARN  Honeycomb rate limit nearly exhausted remaining=1 reset_in_seconds=45
```

---

## Cache invalidation

The cache resets when:
- Grafana restarts or the plugin is reloaded
- The data source is deleted and recreated (new instance = new cache)

To force a cache refresh for a specific panel without restarting:
1. Open the panel edit view
2. Change any query parameter (e.g., add a space to an alias field, then remove it)
3. Run the query — this produces a new execKey, bypassing L2/L3
4. Undo the temporary change

For production use, the 24-hour L3 TTL means that a query result won't change during the day regardless of what Honeycomb receives after the initial load. This is intentional for stable dashboards. If you need fresher results, reduce the TTL (currently a compile-time constant in `pkg/cache/cache.go`).

---

## Limitations of in-process cache

- **Not distributed:** Each Grafana replica maintains its own cache. In a multi-replica Grafana setup, cold start behaviour is per-replica.
- **Lost on restart:** Cache is in-memory only; Grafana restarts flush it.
- **Memory usage:** Each cached result is the full Honeycomb response JSON (typically 10–500 KB). With 1000 unique queries, expect 10–500 MB of cache memory. The janitor runs every 5 minutes to evict expired entries.
