# ADR-004: Deep-Link and Exemplar Strategy

**Status:** Accepted  
**Date:** 2025-01-15

## Context

Users need to pivot from a Grafana visualization to the corresponding Honeycomb query results page for deeper investigation. Honeycomb's Get Query Result response includes a `links.query_url` field pointing directly to the result in the Honeycomb UI.

Grafana supports several mechanism for cross-system linking:
1. **DataLinks** on fields: open URL in new tab; can use field values as template variables
2. **Panel links**: static URL shown on the panel
3. **Exemplars**: Grafana-native exemplar support with a dedicated exemplar frame type
4. **Annotations**: time-range overlays

True Grafana exemplar semantics require a specific frame structure (exemplar frame type, separate from the data frame) and are designed for trace-level sampling data (OpenTelemetry exemplars linking metric data points to specific trace IDs). Honeycomb does not expose trace IDs in query result series data in a way that maps cleanly to Grafana exemplar semantics.

## Decision

### DataLinks (primary mechanism)

Attach a **DataLink** to every data field in every returned frame. The link:
- Opens `links.query_url` from the Honeycomb query result response in a new tab
- The URL already points to the correct dataset, time range, and query result in the Honeycomb UI
- For grouped queries, include the group label values as URL fragments if the Honeycomb URL supports it (it does; Honeycomb's query_url is a permanent result URL)

Implementation: in `transform/frames.go`, after building each frame, call `frame.Fields[i].Config.Links = []data.DataLink{{Title: "Open in Honeycomb", URL: queryURL, TargetBlank: true}}` on every metric field.

### Frame metadata (secondary mechanism)

Attach `frame.Meta.Custom = map[string]interface{}{"honeycombQueryURL": queryURL}` so that panel plugins or Grafana extensions that read frame metadata can also surface the link.

### Panel-level link

The query editor surface a "View in Honeycomb" link when the last result URL is available. This is done via a custom frame meta field that the frontend reads and displays as a contextual link below the query editor.

### Exemplar tradeoff (documented, not implemented)

True Grafana exemplars are not implemented because:
1. Honeycomb series data does not include trace/span IDs in query result rows
2. The exemplar frame format is designed for "specific event linked to a metric data point" semantics, not "time series result linked to a query result page"
3. Implementing fake exemplars with no trace ID would violate Grafana UX expectations

**Future work:** If Honeycomb adds a query API that returns individual event IDs or trace IDs alongside aggregated data, a proper exemplar frame can be added.

### Annotations

Annotation queries are implemented as a special query mode that returns annotation-compatible frames. The annotation body contains the Honeycomb query_url as a clickable link.

## Consequences

**Positive:**
- Every panel always has a direct deep link to the Honeycomb results page
- No special user configuration required
- Links are stable (Honeycomb's query_url is persistent)

**Negative:**
- DataLinks appear on every field; heavy-breakdown queries may generate a noisy tooltip
- Link opens to the full result, not a specific time-point or group row within the result
