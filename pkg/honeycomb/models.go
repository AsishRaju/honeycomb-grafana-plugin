// Package honeycomb provides a typed client for the Honeycomb Query API and
// Query Data API. It does NOT interpret results; callers in the transform
// package are responsible for mapping Honeycomb responses to Grafana frames.
package honeycomb

import (
	"encoding/json"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Query API – Create Query request / response
// ---------------------------------------------------------------------------

// Query is the payload sent to POST /1/queries/{dataset}.
// Fields mirror the Honeycomb API exactly; zero values are omitted via
// `omitempty` so the payload stays minimal.
type Query struct {
	Calculations            []Calculation    `json:"calculations,omitempty"`
	Breakdowns              []string         `json:"breakdowns,omitempty"`
	Filters                 []Filter         `json:"filters,omitempty"`
	FilterCombination       string           `json:"filter_combination,omitempty"`
	Formulas                []Formula        `json:"formulas,omitempty"`
	Havings                 []Having         `json:"havings,omitempty"`
	Orders                  []Order          `json:"orders,omitempty"`
	Limit                   int              `json:"limit,omitempty"`
	Granularity             int              `json:"granularity,omitempty"`
	StartTime               int64            `json:"start_time,omitempty"`
	EndTime                 int64            `json:"end_time,omitempty"`
	TimeRange               int              `json:"time_range,omitempty"`
	CompareTimeOffsetSeconds int             `json:"compare_time_offset_seconds,omitempty"`
	CalculatedFields        []CalculatedField `json:"calculated_fields,omitempty"`
}

// Calculation describes a single aggregation operation.
type Calculation struct {
	Op      string `json:"op"`
	Column  string `json:"column,omitempty"`
	Alias   string `json:"alias,omitempty"` // appears as the column name in results
}

// Filter restricts the events included in the query.
type Filter struct {
	Column string      `json:"column"`
	Op     string      `json:"op"`
	Value  interface{} `json:"value,omitempty"`
}

// Formula is a mathematical expression referencing named calculations.
type Formula struct {
	Alias      string `json:"alias,omitempty"`
	Expression string `json:"expression"`
}

// Having is a post-aggregation filter.
type Having struct {
	CalculateOp string      `json:"calculate_op,omitempty"`
	Column      string      `json:"column,omitempty"`
	Op          string      `json:"op"`
	Value       interface{} `json:"value,omitempty"`
}

// Order specifies sort terms for results.
type Order struct {
	Op     string `json:"op,omitempty"`
	Column string `json:"column,omitempty"`
	Order  string `json:"order"` // "ascending" or "descending"
}

// CalculatedField is a derived property definition.
type CalculatedField struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// QueryResponse is returned by POST /1/queries/{dataset} on success.
type QueryResponse struct {
	ID string `json:"id"`
	// The API echoes back all query fields; we only need the ID for subsequent calls.
}

// ---------------------------------------------------------------------------
// Query Data API – Create Query Result request / response
// ---------------------------------------------------------------------------

// QueryResultRequest is the payload for POST /1/query_results/{dataset}.
type QueryResultRequest struct {
	QueryID                 string `json:"query_id"`
	DisableSeries           bool   `json:"disable_series,omitempty"`
	DisableTotalByAggregate bool   `json:"disable_total_by_aggregate,omitempty"`
	DisableOtherByAggregate bool   `json:"disable_other_by_aggregate,omitempty"`
	Limit                   int    `json:"limit,omitempty"`
}

// QueryResultCreateResponse is the 201 response from Create Query Result.
type QueryResultCreateResponse struct {
	ID       string `json:"id"`
	Complete bool   `json:"complete"`
	Links    Links  `json:"links"`
}

// ---------------------------------------------------------------------------
// Query Data API – Get Query Result response
// ---------------------------------------------------------------------------

// QueryResultResponse is the response from GET /1/query_results/{dataset}/{id}.
type QueryResultResponse struct {
	ID       string      `json:"id"`
	Complete bool        `json:"complete"`
	Query    Query       `json:"query"`
	Data     *ResultData `json:"data,omitempty"`
	Links    Links       `json:"links"`
}

// ResultData carries the actual query output.
type ResultData struct {
	// Series is timeseries data: one element per (time_bucket × group) tuple.
	// Each element has a Unix timestamp and a map of column→value pairs that
	// includes both breakdown column values and calculation results.
	Series []SeriesEntry `json:"series"`
	// Results is the summary (non-timeseries) aggregation: one element per group.
	Results []ResultEntry `json:"results"`
	// TotalByAggregate is the total across all groups for each calculation.
	TotalByAggregate map[string]interface{} `json:"total_by_aggregate,omitempty"`
	// OtherByAggregate is the aggregate for rows excluded by the limit.
	OtherByAggregate map[string]interface{} `json:"other_by_aggregate,omitempty"`
}

// SeriesEntry is one row in the timeseries response.
// Data keys are either breakdown column names or calculation names
// (e.g. "COUNT", "AVG(duration_ms)").
type SeriesEntry struct {
	Time FlexibleTime           `json:"time"`
	Data map[string]interface{} `json:"data"`
}

// FlexibleTime handles Honeycomb's time field which may be a string
// (ISO 8601 like "2024-01-15T12:00:00Z") or a number (Unix epoch seconds).
type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	// Try as number (Unix epoch seconds)
	if len(s) > 0 && s[0] != '"' {
		var epoch int64
		if err := json.Unmarshal(b, &epoch); err == nil {
			ft.Time = time.Unix(epoch, 0).UTC()
			return nil
		}
	}
	// Try as string (ISO 8601)
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return fmt.Errorf("cannot parse time %q: not a number or string", s)
	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// Try without timezone
		t, err = time.Parse("2006-01-02T15:04:05", str)
		if err != nil {
			return fmt.Errorf("cannot parse time %q: %w", str, err)
		}
		t = t.UTC()
	}
	ft.Time = t
	return nil
}

// ResultEntry is one row in the summary results response.
// Keys are the same as SeriesEntry.Data.
type ResultEntry map[string]interface{}

// Links carries UI URLs returned by Honeycomb.
type Links struct {
	QueryURL      string `json:"query_url"`
	GraphImageURL string `json:"graph_image_url"`
}

// ---------------------------------------------------------------------------
// Rate limit headers
// ---------------------------------------------------------------------------

// RateLimitInfo is parsed from the RateLimit response header.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Duration // time until window reset
}
