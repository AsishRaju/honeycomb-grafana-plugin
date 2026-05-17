# Honeycomb Grafana Data Source Plugin

A production-grade Grafana backend data source plugin for [Honeycomb](https://www.honeycomb.io) that lets you query Honeycomb datasets directly from Grafana dashboards.

[![CI](https://github.com/honeycombio/grafana-honeycomb-datasource/actions/workflows/ci.yml/badge.svg)](https://github.com/honeycombio/grafana-honeycomb-datasource/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

---

## Features

- **Full Honeycomb Query API support** — calculations, filters, group-by (breakdowns), order-by, limit, granularity, compare time offset
- **Three panel types** — timeseries, table, and stat
- **First-class high-cardinality group-by** — each breakdown combination becomes a separate Grafana series; series are labeled, ordered, and limited with guardrails
- **Aggressive caching** — three-level cache (query_id → result_id → completed result) protects against Honeycomb's 10 req/min Create Query Result limit
- **Token-bucket rate limiter** — 8 tokens/60 s with automatic exponential backoff on 429
- **Singleflight deduplication** — concurrent panels requesting identical data collapse into a single Honeycomb API call
- **Deep links** — every metric field carries a DataLink opening the corresponding Honeycomb query result page in a new tab
- **Dashboard variables** — list datasets or columns; use the result as template variables across all panels
- **Annotation support** — use Honeycomb queries as annotation sources
- **Health check** — validates API key and connectivity from Grafana's data source settings page
- **Secure secret handling** — API key is stored in Grafana's `secureJsonData` (encrypted at rest) and never sent to the browser
- **Raw JSON mode** — power users can paste raw Honeycomb Query API JSON
- **US and EU region support** — configurable API base URL

---

## Requirements

- Grafana ≥ 11.0.0
- A Honeycomb Configuration API key with **Manage Queries and Columns** and **Run Queries** permissions

---

## Installation

### From GitHub Releases (recommended)

1. Download the latest release zip from [Releases](https://github.com/honeycombio/grafana-honeycomb-datasource/releases).
2. Extract to your Grafana plugins directory:
   ```bash
   unzip honeycombio-honeycomb-datasource-0.1.0.zip -d /var/lib/grafana/plugins/
   ```
3. Restart Grafana.
4. If the plugin is not signed, add to `grafana.ini`:
   ```ini
   [plugins]
   allow_loading_unsigned_plugins = honeycombio-honeycomb-datasource
   ```

### Via Grafana CLI

```bash
grafana-cli plugins install honeycombio-honeycomb-datasource
```

### Via provisioning

See [provisioning/datasources/honeycomb.yaml](provisioning/datasources/honeycomb.yaml).

---

## Configuration

1. In Grafana, go to **Settings → Data sources → Add data source**.
2. Search for **Honeycomb** and select it.
3. Configure:
   - **API Region**: US (default) or EU; or enter a custom URL
   - **API Key**: your Honeycomb Configuration API key (stored encrypted)
4. Click **Save & test** to verify connectivity.

---

## Quick start: first query

1. Create a new dashboard and add a **Time series** panel.
2. Select the **Honeycomb** datasource.
3. Select your **Dataset**.
4. Add a **Calculation**: `COUNT`.
5. Add a **Group by** column: `service.name`.
6. Set **Limit** to 10 and **Order by** `COUNT` descending.
7. Click **Run query**.

Each `service.name` value will appear as a separate time series. Click any data point and then **Open in Honeycomb** to jump directly to the corresponding Honeycomb query result.

---

## How caching works

Honeycomb limits Create Query Result to **10 requests/minute**. The plugin uses a three-level cache to minimize these calls:

| Level | Caches | TTL | Benefit |
|-------|--------|-----|---------|
| L1 | Query spec → `query_id` | 1 hour | Reuse the same Honeycomb query across time range changes |
| L2 | Execution context → `query_result_id` | 30 min | Skip re-submission for recently-run queries |
| L3 | `query_result_id` → completed result | 24 hours | Serve identical queries from memory without any Honeycomb call |

On a steady-state dashboard, panel refreshes almost always hit L3 (in-memory, microsecond latency). Only cold cache loads or new time ranges hit the rate-limited Create Query Result endpoint.

See [ADR-002](docs/adr/ADR-002-caching-strategy.md) for full details.

---

## Dashboard variables

Use the Honeycomb datasource for dashboard variable queries:

| Variable type | Query | Returns |
|--------------|-------|---------|
| Datasets | `{ "queryType": "datasets" }` | All dataset slug/name pairs |
| Columns | `{ "queryType": "columns", "dataset": "production" }` | Column names for a dataset |

Variables can be used in any string field: `dataset: $dataset`, `breakdowns: ["$column"]`.

---

## Deep links

Every panel backed by this datasource includes a **"Open in Honeycomb"** link on each data series. Clicking the link opens the corresponding Honeycomb query result page in a new browser tab. The link is attached to every numeric field in every returned frame.

See [ADR-004](docs/adr/ADR-004-deep-links.md) for the design rationale.

---

## Known limitations

- Cache is in-process; it resets on Grafana restart and is not shared across Grafana replicas.
- Compare time offset (`compare_time_offset_seconds`) submits the comparison window, but Honeycomb's current API does not return comparison data in query result responses.
- Exemplar frames (native Grafana exemplar semantics) are not implemented; deep links via DataLinks are used instead. See [ADR-004](docs/adr/ADR-004-deep-links.md).
- Maximum 7 days of query history (Honeycomb API limit).
- Granularity must be within Honeycomb's valid range: `time_range/1000` to `time_range/1` seconds.

---

## Local development

See [CONTRIBUTING.md](CONTRIBUTING.md) for full setup instructions.

```bash
# Build backend + frontend, start Grafana:
mage build:darwin && npm run build
docker-compose up
# Open http://localhost:3000
```

---

## Architecture

See the [Architecture Decision Records](docs/adr/) for the major design decisions:

- [ADR-001](docs/adr/ADR-001-backend-plugin.md) — Why a backend plugin
- [ADR-002](docs/adr/ADR-002-caching-strategy.md) — Multi-level caching strategy
- [ADR-003](docs/adr/ADR-003-rate-limiting.md) — Rate limiting strategy
- [ADR-004](docs/adr/ADR-004-deep-links.md) — Deep link and exemplar strategy

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
