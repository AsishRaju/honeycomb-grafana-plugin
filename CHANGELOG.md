# Changelog

All notable changes to the Honeycomb Grafana Data Source Plugin are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.4-etsy] — 2026-07-27

Etsy fork build bundling the local production fixes on top of upstream `0.1.2`.

### Fixed
- Normalize visual-editor `in` / `not-in` filter values from CSV strings into JSON arrays before calling Honeycomb (Metrics, Logs, Trace Search), avoiding 422s from scalar payloads.

### Added
- **Open in Honeycomb** button in the query editor, using the backend `honeycombQueryURL` frame meta after a successful run.

### Changed
- Raise the local Create Query Result token bucket to **60 req/min** with burst **30** for denser Grafana dashboards (above Honeycomb's published ~10/min hard cap; 429 handling still applies).

## [0.1.0] — 2025-01-15

### Added
- Initial release of the Honeycomb Grafana data source plugin.
- Backend plugin (Go) with full Honeycomb Query API and Query Data API support.
- Three-level in-process cache (query_id → result_id → completed result) to protect against the 10 req/min rate limit on Create Query Result.
- Token-bucket rate limiter at 8 tokens/60 s with exponential backoff on 429.
- Singleflight deduplication to prevent concurrent dashboard panels from fanning out identical Honeycomb API calls.
- Support for timeseries, table, and stat query modes.
- High-cardinality group-by / breakdown support: one Grafana frame per breakdown group in timeseries mode.
- Deep links to Honeycomb query result pages, attached as DataLinks on every metric field.
- Dashboard variable support: list datasets and list columns for a dataset.
- Annotation query support.
- Health check endpoint.
- Visual query editor with dataset selector, calculations editor, filters editor, group-by editor, order-by editor.
- Raw JSON mode for power users who want direct access to the Honeycomb Query API.
- Template variable substitution in all string query fields.
- Configurable API base URL for US and EU Honeycomb accounts.
- Secure API key storage in Grafana `secureJsonData` (encrypted at rest; never sent to browser).
- Architecture Decision Records (ADR-001 through ADR-004).
- Docker Compose local development environment.
- Example provisioning configuration and example dashboard.
- Apache 2.0 license.

[Unreleased]: https://github.com/honeycombio/grafana-honeycomb-datasource/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/honeycombio/grafana-honeycomb-datasource/releases/tag/v0.1.0
