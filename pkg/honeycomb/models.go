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
//
// Filters and FilterCombination are per-calculation filters that scope this
// calc's input rows. Honeycomb requires the Metrics Beta feature flag for
// this to work; queries will be rejected with a 4xx otherwise.
//
// See https://docs.honeycomb.io/api/queries/create-a-query.md
type Calculation struct {
	Op                string   `json:"op"`
	Column            string   `json:"column,omitempty"`
	Alias             string   `json:"alias,omitempty"` // appears as the column name in results
	Filters           []Filter `json:"filters,omitempty"`
	FilterCombination string   `json:"filter_combination,omitempty"`
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
//
// Honeycomb's wire format wraps each row under a "data" key
// (e.g. `{"data": {"COUNT": 42, "trace.trace_id": "..."}}`); we keep
// ResultEntry flat in memory and unwrap during decode so transformers
// stay simple. Flat-shaped JSON also decodes cleanly so existing tests
// and any caller that builds entries directly continue to work.
type ResultEntry map[string]interface{}

// UnmarshalJSON unwraps the Honeycomb {"data": {...}} envelope when
// present. Falls back to flat decode otherwise.
func (r *ResultEntry) UnmarshalJSON(b []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	if dataRaw, ok := fields["data"]; ok {
		var data map[string]interface{}
		if err := json.Unmarshal(dataRaw, &data); err != nil {
			return fmt.Errorf("ResultEntry: decode data field: %w", err)
		}
		*r = ResultEntry(data)
		return nil
	}
	var flat map[string]interface{}
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*r = ResultEntry(flat)
	return nil
}

// Links carries UI URLs returned by Honeycomb.
type Links struct {
	QueryURL      string `json:"query_url"`
	GraphImageURL string `json:"graph_image_url"`
}

// ---------------------------------------------------------------------------
// SLOs API
// ---------------------------------------------------------------------------

// SLO is the Honeycomb Service Level Objective shape returned by
// GET /1/slos/{datasetSlug} and GET /1/slos/{datasetSlug}/{sloId}.
//
// The "detailed" fields (Compliance, BudgetRemaining, Status, BurnRate) are
// only populated when GetSLO is called with detailed=true. ListSLOs always
// omits them per the Honeycomb spec.
//
// See https://docs.honeycomb.io/api/slos/get-an-slo.md
type SLO struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description,omitempty"`
	SLI              SLI        `json:"sli"`
	TimePeriodDays   int        `json:"time_period_days"`
	TargetPerMillion int        `json:"target_per_million"`
	Tags             []SLOTag   `json:"tags,omitempty"`
	ResetAt          *time.Time `json:"reset_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at,omitempty"`
	DatasetSlugs     []string   `json:"dataset_slugs,omitempty"`

	// Detailed-only fields (present when ?detailed=true on GetSLO).
	Compliance      *float64 `json:"compliance,omitempty"`
	BudgetRemaining *float64 `json:"budget_remaining,omitempty"`
	Status          string   `json:"status,omitempty"`
	BurnRate        *float64 `json:"burn_rate,omitempty"`
}

// SLI is the Service Level Indicator that drives an SLO; it references a
// derived (calculated) field by alias.
type SLI struct {
	Alias string `json:"alias"`
}

// SLOTag is one of the up-to-10 key/value tags on an SLO.
type SLOTag struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
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
