package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeycombio/grafana-honeycomb-datasource/pkg/honeycomb"
	"github.com/honeycombio/grafana-honeycomb-datasource/pkg/transform"
)

// HoneycombQuery is the deserialized form of a Grafana panel query for
// this datasource. It maps to the query editor's state in the frontend.
type HoneycombQuery struct {
	// Dataset is the Honeycomb dataset slug (required).
	Dataset string `json:"dataset"`

	// QueryMode controls how results are mapped to Grafana frames.
	// Defaults to "timeseries".
	QueryMode string `json:"queryMode"` // "timeseries" | "table" | "stat"

	// Calculations lists the aggregation operations to compute.
	Calculations []Calculation `json:"calculations"`

	// Filters restricts events included in the query.
	Filters []Filter `json:"filters"`

	// FilterCombination is "AND" or "OR" (default: "AND").
	FilterCombination string `json:"filterCombination"`

	// Breakdowns lists column names to group by.
	Breakdowns []string `json:"breakdowns"`

	// Orders controls result sorting.
	Orders []Order `json:"orders"`

	// Limit caps the number of result groups (default 100, max 10000).
	Limit int `json:"limit"`

	// Granularity is the time resolution in seconds (0 = auto-derive).
	Granularity int `json:"granularity"`

	// CompareTimeOffset adds a historical comparison in seconds.
	// Valid values: 1800, 3600, 7200, 28800, 86400, 604800, 2419200, 15724800.
	CompareTimeOffset int `json:"compareTimeOffset"`

	// RawMode bypasses the query editor and sends RawJSON directly.
	RawMode bool `json:"rawMode"`

	// RawJSON is used when RawMode is true. Must be a valid Honeycomb Query JSON.
	RawJSON string `json:"rawJson"`
}

// Calculation represents one aggregation operation in the query.
type Calculation struct {
	Op     string `json:"op"`
	Column string `json:"column,omitempty"`
	Alias  string `json:"alias,omitempty"`
}

// Filter restricts events in the query.
type Filter struct {
	Column string      `json:"column"`
	Op     string      `json:"op"`
	Value  interface{} `json:"value,omitempty"`
}

// Order specifies a sort term.
type Order struct {
	Op     string `json:"op,omitempty"`
	Column string `json:"column,omitempty"`
	Order  string `json:"order"` // "ascending" | "descending"
}

// Validate checks that the query has all required fields and returns an error
// with a descriptive message if any required field is missing or invalid.
func (q *HoneycombQuery) Validate() error {
	if strings.TrimSpace(q.Dataset) == "" {
		return fmt.Errorf("dataset is required")
	}
	if q.RawMode {
		if strings.TrimSpace(q.RawJSON) == "" {
			return fmt.Errorf("rawJson is required when rawMode is true")
		}
		var raw honeycomb.Query
		if err := json.Unmarshal([]byte(q.RawJSON), &raw); err != nil {
			return fmt.Errorf("rawJson is not valid Honeycomb query JSON: %w", err)
		}
	} else if len(q.Calculations) == 0 {
		return fmt.Errorf("at least one calculation is required")
	}
	if q.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	if q.Limit > 10000 {
		return fmt.Errorf("limit cannot exceed 10000")
	}
	return nil
}

// IsEmpty returns true if the query has no meaningful content and should be
// skipped. This prevents sending empty queries to Honeycomb when a panel is
// freshly added to a dashboard.
func (q *HoneycombQuery) IsEmpty() bool {
	return strings.TrimSpace(q.Dataset) == "" &&
		len(q.Calculations) == 0 &&
		!q.RawMode
}

// ToHoneycombQuery converts a plugin query into the Honeycomb API format.
// Time range is NOT included here; it is applied separately via
// fingerprint.ApplyTimeRange so the cache key remains time-independent at L1.
func (q *HoneycombQuery) ToHoneycombQuery() (honeycomb.Query, error) {
	if q.RawMode {
		var raw honeycomb.Query
		if err := json.Unmarshal([]byte(q.RawJSON), &raw); err != nil {
			return honeycomb.Query{}, fmt.Errorf("parse raw query: %w", err)
		}
		return raw, nil
	}

	hq := honeycomb.Query{
		Breakdowns:        q.Breakdowns,
		FilterCombination: q.FilterCombination,
		Granularity:       q.Granularity,
		Limit:             q.Limit,
	}

	// Calculations
	hq.Calculations = make([]honeycomb.Calculation, len(q.Calculations))
	for i, c := range q.Calculations {
		hq.Calculations[i] = honeycomb.Calculation{
			Op:     c.Op,
			Column: c.Column,
			Alias:  c.Alias,
		}
	}

	// Filters
	hq.Filters = make([]honeycomb.Filter, len(q.Filters))
	for i, f := range q.Filters {
		hq.Filters[i] = honeycomb.Filter{
			Column: f.Column,
			Op:     f.Op,
			Value:  f.Value,
		}
	}

	// Orders
	hq.Orders = make([]honeycomb.Order, len(q.Orders))
	for i, o := range q.Orders {
		hq.Orders[i] = honeycomb.Order{
			Op:     o.Op,
			Column: o.Column,
			Order:  o.Order,
		}
	}

	// Compare time offset (optional).
	if q.CompareTimeOffset > 0 {
		hq.CompareTimeOffsetSeconds = q.CompareTimeOffset
	}

	return hq, nil
}

// QueryMode maps the query's string mode to the transform.QueryMode enum.
func (q *HoneycombQuery) FrameMode() transform.QueryMode {
	switch q.QueryMode {
	case "table":
		return transform.ModeTable
	case "stat":
		return transform.ModeStat
	default:
		return transform.ModeTimeseries
	}
}

// ShouldDisableSeries returns true when the query mode does not need timeseries
// data. Sending disable_series=true to Honeycomb reduces response payload and
// unlocks higher result limits.
func (q *HoneycombQuery) ShouldDisableSeries() bool {
	switch q.QueryMode {
	case "table", "stat":
		return true
	default:
		return false
	}
}

// DefaultQuery returns a minimal valid query to show as a starting point
// when a user adds a new Honeycomb panel.
func DefaultQuery() HoneycombQuery {
	return HoneycombQuery{
		QueryMode:         "timeseries",
		Calculations:      []Calculation{{Op: "COUNT"}},
		FilterCombination: "AND",
		Limit:             100,
	}
}

// parseQuery deserializes the JSON from a Grafana backend.DataQuery.JSON field.
func parseQuery(rawJSON []byte) (HoneycombQuery, error) {
	var q HoneycombQuery
	if len(rawJSON) == 0 {
		return DefaultQuery(), nil
	}
	if err := json.Unmarshal(rawJSON, &q); err != nil {
		return HoneycombQuery{}, fmt.Errorf("parse query JSON: %w", err)
	}
	// Apply defaults.
	if q.QueryMode == "" {
		q.QueryMode = "timeseries"
	}
	if q.FilterCombination == "" {
		q.FilterCombination = "AND"
	}
	if q.Limit == 0 {
		q.Limit = 100
	}
	return q, nil
}
