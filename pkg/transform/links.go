// Package transform converts Honeycomb API responses into Grafana DataFrames
// and attaches deep links back to the Honeycomb UI.
package transform

import (
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Trace ID column names recognized for deep linking. Honeycomb's standard
// OpenTelemetry semantic convention is "trace.trace_id"; legacy/custom
// instrumentations sometimes use "trace_id".
var traceIDColumns = map[string]struct{}{
	"trace.trace_id": {},
	"trace_id":       {},
}

// AttachDeepLink adds a DataLink to every metric field in frame that opens
// the given Honeycomb query result URL in a new browser tab.
//
// The link is attached to metric fields only (not time or string/label
// breakdown fields) to keep the Grafana tooltip clean.
//
// See ADR-004 for the deep-link strategy.
func AttachDeepLink(frame *data.Frame, queryURL string) {
	if queryURL == "" || frame == nil {
		return
	}
	link := data.DataLink{
		Title:       "Open in Honeycomb",
		URL:         queryURL,
		TargetBlank: true,
	}
	for _, field := range frame.Fields {
		if field == nil {
			continue
		}
		// Attach links only to numeric (metric) fields, not time or string fields.
		switch field.Type() {
		case data.FieldTypeFloat64, data.FieldTypeNullableFloat64,
			data.FieldTypeInt64, data.FieldTypeNullableInt64,
			data.FieldTypeUint64, data.FieldTypeNullableUint64:
			if field.Config == nil {
				field.Config = &data.FieldConfig{}
			}
			field.Config.Links = append(field.Config.Links, link)
		}
	}
}

// AttachTraceLinks scans frame for fields named "trace.trace_id" or
// "trace_id" and adds a "Open trace in Honeycomb" DataLink to each one.
//
// The link template substitutes the row's trace_id value via Grafana's
// ${__value.raw} expression so each row gets its own clickable URL.
//
// If team is empty, no link is attached (we cannot construct a stable UI URL
// without it). Environment may be empty for Classic accounts; the URL omits
// the /environments/<env> path segment in that case.
//
// apiURL is used to derive the UI URL — api.honeycomb.io ↔ ui.honeycomb.io,
// api.eu1.honeycomb.io ↔ ui.eu1.honeycomb.io.
func AttachTraceLinks(frame *data.Frame, apiURL, team, environment, dataset string) {
	if frame == nil || team == "" || dataset == "" {
		return
	}
	uiBase := uiBaseURL(apiURL)
	traceURLTemplate := buildTraceURLTemplate(uiBase, team, environment, dataset)

	link := data.DataLink{
		Title:       "Open trace in Honeycomb",
		URL:         traceURLTemplate,
		TargetBlank: true,
	}

	for _, field := range frame.Fields {
		if field == nil {
			continue
		}
		if _, ok := traceIDColumns[field.Name]; !ok {
			continue
		}
		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.Links = append(field.Config.Links, link)
	}
}

// uiBaseURL derives the Honeycomb UI base URL from the API base URL.
// Falls back to https://ui.honeycomb.io when the input cannot be mapped.
func uiBaseURL(apiURL string) string {
	const defaultUI = "https://ui.honeycomb.io"
	if apiURL == "" {
		return defaultUI
	}
	parsed, err := url.Parse(strings.TrimRight(apiURL, "/"))
	if err != nil || parsed.Host == "" {
		return defaultUI
	}
	host := parsed.Host
	// api.honeycomb.io      → ui.honeycomb.io
	// api.eu1.honeycomb.io  → ui.eu1.honeycomb.io
	// api-foo.honeycomb.io  → ui-foo.honeycomb.io (best-effort)
	if strings.HasPrefix(host, "api.") {
		host = "ui." + strings.TrimPrefix(host, "api.")
	} else if strings.HasPrefix(host, "api-") {
		host = "ui-" + strings.TrimPrefix(host, "api-")
	}
	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + host
}

// buildTraceURLTemplate constructs the Honeycomb trace deep-link URL with a
// Grafana ${__value.raw} placeholder for the trace_id.
func buildTraceURLTemplate(uiBase, team, environment, dataset string) string {
	team = url.PathEscape(team)
	dataset = url.PathEscape(dataset)
	if environment == "" {
		// Classic account URL form.
		return uiBase + "/" + team + "/datasets/" + dataset + "/trace?trace_id=${__value.raw}"
	}
	environment = url.PathEscape(environment)
	return uiBase + "/" + team + "/environments/" + environment + "/datasets/" + dataset + "/trace?trace_id=${__value.raw}"
}

// SetFrameMeta attaches the Honeycomb query URL to the frame's custom
// metadata so panel plugins and extensions can surface it independently.
func SetFrameMeta(frame *data.Frame, queryURL, graphImageURL string) {
	if frame == nil {
		return
	}
	custom := map[string]interface{}{}
	if queryURL != "" {
		custom["honeycombQueryURL"] = queryURL
	}
	if graphImageURL != "" {
		custom["honeycombGraphImageURL"] = graphImageURL
	}
	if len(custom) > 0 {
		if frame.Meta == nil {
			frame.Meta = &data.FrameMeta{}
		}
		frame.Meta.Custom = custom
	}
}
