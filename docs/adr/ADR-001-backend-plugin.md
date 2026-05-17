# ADR-001: Backend Plugin Architecture

**Status:** Accepted  
**Date:** 2025-01-15  
**Author:** Engineering Team

## Context

Grafana data source plugins can be implemented as frontend-only (JavaScript communicates directly with the external API via Grafana's proxy) or as full-stack plugins (a Go backend component handles all external API communication).

Honeycomb requires an API key for all requests. That key must never be exposed to the browser. Additionally, the plugin needs:
- Server-side caching shared across dashboard panels and users
- A token-bucket rate limiter protecting against the 10 req/min Create Query Result hard cap
- In-flight deduplication (singleflight) so a busy dashboard with many panels doesn't fan out N identical Honeycomb calls
- Structured metrics, logs, and traces for operator observability

None of these are possible in a frontend-only plugin.

## Decision

Build as a **Grafana backend plugin** with a Go backend component and TypeScript frontend.

The frontend is responsible only for:
- Rendering the query editor UI
- Applying template variable substitution before sending queries to the backend
- Filtering out empty/hidden queries before they reach the backend

The backend is responsible for:
- All Honeycomb API communication
- Secret (API key) management via `secureJsonData`
- Query normalization and fingerprinting
- Multi-level caching (query_id → result_id → completed result)
- Rate limiting (token bucket, 8 tokens/60 s)
- In-flight deduplication (singleflight)
- Frame transformation (Honeycomb response → Grafana DataFrames)
- Deep-link attachment
- Resource handlers for metadata (datasets, columns)
- Health checks

## Consequences

**Positive:**
- API key never reaches the browser
- Shared cache across all panels and users on the same Grafana instance
- Rate limiter protects against dashboard stampede
- Full observability instrumentation available in Go
- Alerting support works out of the box (`"alerting": true` in plugin.json)

**Negative:**
- Requires deploying the Go binary alongside the Grafana process
- Binary must be compiled per OS/architecture (handled by the release workflow)
- Slightly more complex local development setup (requires `mage` and Go toolchain)

## Alternatives Rejected

**Frontend-only with Grafana proxy:** Cannot share server-side cache or rate limiter. API key visible in browser network requests. No singleflight. Rejected.

**Frontend-only with per-panel caching:** Caches are isolated per-panel and reset on page load. Provides no protection against the 10 req/min limit across panels. Rejected.
