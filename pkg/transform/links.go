// Package transform converts Honeycomb API responses into Grafana DataFrames
// and attaches deep links back to the Honeycomb UI.
package transform

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

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
