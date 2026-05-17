package transform

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/honeycombio/grafana-honeycomb-datasource/pkg/honeycomb"
)

// QueryMode controls how the Honeycomb result is mapped to Grafana frames.
type QueryMode string

const (
	// ModeTimeseries produces one frame per breakdown group, each with a time
	// column and one column per calculation. Use for time series panels.
	ModeTimeseries QueryMode = "timeseries"

	// ModeTable produces a single frame with all breakdown and calculation
	// columns as rows. Use for Table panels.
	ModeTable QueryMode = "table"

	// ModeStat produces a single frame with a single numeric value.
	// Picks the first calculation result from the first (or only) group.
	ModeStat QueryMode = "stat"
)

// FrameOptions controls frame generation.
type FrameOptions struct {
	Mode        QueryMode
	QueryURL    string // Honeycomb result deep link
	GraphURL    string
	// MaxGroups limits the number of series in timeseries mode to prevent
	// UI overload. 0 means use DefaultMaxGroups.
	MaxGroups   int
}

const DefaultMaxGroups = 500

// ToFrames converts a Honeycomb QueryResultResponse to Grafana DataFrames.
//
// For timeseries mode: returns one frame per breakdown group. Each frame has
// a time field and one field per calculation. Labels are set to the breakdown
// column values for that group.
//
// For table mode: returns a single frame with breakdown columns + calculation
// columns as fields, one row per result entry.
//
// For stat mode: returns a single frame with a single value from the first
// calculation of the first result row.
func ToFrames(result *honeycomb.QueryResultResponse, opts FrameOptions) (data.Frames, error) {
	if result == nil || result.Data == nil {
		return data.Frames{emptyFrame(opts)}, nil
	}

	maxGroups := opts.MaxGroups
	if maxGroups <= 0 {
		maxGroups = DefaultMaxGroups
	}

	switch opts.Mode {
	case ModeTable:
		return toTableFrames(result, opts)
	case ModeStat:
		return toStatFrames(result, opts)
	default:
		return toTimeseriesFrames(result, opts, maxGroups)
	}
}

// ---------------------------------------------------------------------------
// Timeseries
// ---------------------------------------------------------------------------

// toTimeseriesFrames converts the series data into per-group Grafana frames.
func toTimeseriesFrames(result *honeycomb.QueryResultResponse, opts FrameOptions, maxGroups int) (data.Frames, error) {
	series := result.Data.Series
	if len(series) == 0 {
		return data.Frames{emptyFrame(opts)}, nil
	}

	// Identify all calculation column names (those that appear as metric values)
	// and all breakdown column names from the query spec.
	calcCols := calcColumnNames(result.Query.Calculations)
	breakdownCols := result.Query.Breakdowns

	// Build a group key → per-time data map.
	// group key is a canonical string encoding the breakdown values.
	type timePoint struct {
		t    time.Time
		vals map[string]float64 // calc column → value
	}
	type group struct {
		labels     data.Labels
		timePoints []timePoint
	}
	groups := make(map[string]*group)
	var groupOrder []string // for deterministic frame ordering

	for _, entry := range series {
		t := entry.Time.UTC()
		labels := extractBreakdowns(entry.Data, breakdownCols)
		key := groupKey(labels)

		g, ok := groups[key]
		if !ok {
			if len(groups) >= maxGroups {
				continue // silently drop groups beyond the limit
			}
			g = &group{labels: labels}
			groups[key] = g
			groupOrder = append(groupOrder, key)
		}

		vals := make(map[string]float64, len(calcCols))
		for _, col := range calcCols {
			if v, ok := entry.Data[col]; ok {
				vals[col] = toFloat64(v)
			}
		}
		g.timePoints = append(g.timePoints, timePoint{t: t, vals: vals})
	}

	if len(groups) == 0 {
		return data.Frames{emptyFrame(opts)}, nil
	}

	// Sort groups for a stable frame ordering across refreshes.
	sort.Strings(groupOrder)

	frames := make(data.Frames, 0, len(groups))
	for _, key := range groupOrder {
		g := groups[key]

		// Sort time points within each group.
		sort.Slice(g.timePoints, func(i, j int) bool {
			return g.timePoints[i].t.Before(g.timePoints[j].t)
		})

		timeVals := make([]time.Time, len(g.timePoints))
		for i, tp := range g.timePoints {
			timeVals[i] = tp.t
		}

		frame := data.NewFrame("", data.NewField("time", nil, timeVals))
		frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMany})

		for _, col := range calcCols {
			fieldVals := make([]*float64, len(g.timePoints))
			for i, tp := range g.timePoints {
				if v, ok := tp.vals[col]; ok {
					vv := v
					fieldVals[i] = &vv
				}
			}
			field := data.NewField(col, g.labels, fieldVals)
			if field.Config == nil {
				field.Config = &data.FieldConfig{}
			}
			field.Config.DisplayNameFromDS = labeledName(col, g.labels)
			frame.Fields = append(frame.Fields, field)
		}

		SetFrameMeta(frame, opts.QueryURL, opts.GraphURL)
		AttachDeepLink(frame, opts.QueryURL)
		frames = append(frames, frame)
	}

	return frames, nil
}

// ---------------------------------------------------------------------------
// Table
// ---------------------------------------------------------------------------

func toTableFrames(result *honeycomb.QueryResultResponse, opts FrameOptions) (data.Frames, error) {
	rows := result.Data.Results
	if len(rows) == 0 {
		return data.Frames{emptyFrame(opts)}, nil
	}

	breakdownCols := result.Query.Breakdowns
	calcCols := calcColumnNames(result.Query.Calculations)

	// Build fields.
	stringFields := make(map[string][]string, len(breakdownCols))
	floatFields := make(map[string][]*float64, len(calcCols))

	for _, col := range breakdownCols {
		stringFields[col] = make([]string, 0, len(rows))
	}
	for _, col := range calcCols {
		floatFields[col] = make([]*float64, 0, len(rows))
	}

	for _, row := range rows {
		for _, col := range breakdownCols {
			v := ""
			if raw, ok := row[col]; ok {
				v = fmt.Sprintf("%v", raw)
			}
			stringFields[col] = append(stringFields[col], v)
		}
		for _, col := range calcCols {
			if raw, ok := row[col]; ok {
				f := toFloat64(raw)
				floatFields[col] = append(floatFields[col], &f)
			} else {
				floatFields[col] = append(floatFields[col], nil)
			}
		}
	}

	frame := data.NewFrame("honeycomb")
	frame.Meta = &data.FrameMeta{Type: data.FrameTypeTable}

	for _, col := range breakdownCols {
		frame.Fields = append(frame.Fields, data.NewField(col, nil, stringFields[col]))
	}
	for _, col := range calcCols {
		frame.Fields = append(frame.Fields, data.NewField(col, nil, floatFields[col]))
	}

	SetFrameMeta(frame, opts.QueryURL, opts.GraphURL)
	AttachDeepLink(frame, opts.QueryURL)
	return data.Frames{frame}, nil
}

// ---------------------------------------------------------------------------
// Stat
// ---------------------------------------------------------------------------

func toStatFrames(result *honeycomb.QueryResultResponse, opts FrameOptions) (data.Frames, error) {
	rows := result.Data.Results
	calcCols := calcColumnNames(result.Query.Calculations)

	if len(rows) == 0 || len(calcCols) == 0 {
		return data.Frames{emptyFrame(opts)}, nil
	}

	col := calcCols[0]
	raw, ok := rows[0][col]
	if !ok {
		return data.Frames{emptyFrame(opts)}, nil
	}
	v := toFloat64(raw)

	field := data.NewField(col, nil, []*float64{&v})
	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	frame := data.NewFrame("honeycomb", field)
	SetFrameMeta(frame, opts.QueryURL, opts.GraphURL)
	AttachDeepLink(frame, opts.QueryURL)
	return data.Frames{frame}, nil
}

// ---------------------------------------------------------------------------
// Annotations
// ---------------------------------------------------------------------------

// ToAnnotationFrames converts a Honeycomb result to annotation-compatible frames.
// Each result row becomes an annotation with the query URL as the URL.
func ToAnnotationFrames(result *honeycomb.QueryResultResponse, opts FrameOptions) (data.Frames, error) {
	if result == nil || result.Data == nil {
		return nil, nil
	}

	series := result.Data.Series
	if len(series) == 0 {
		return nil, nil
	}

	times := make([]time.Time, 0, len(series))
	texts := make([]string, 0, len(series))
	urls := make([]string, 0, len(series))
	// tags is a comma-separated string per annotation row (Grafana expects []string).
	tags := make([]string, 0, len(series))

	breakdownCols := result.Query.Breakdowns
	calcCols := calcColumnNames(result.Query.Calculations)

	for _, entry := range series {
		t := entry.Time.UTC()
		times = append(times, t)

		var parts []string
		for _, col := range calcCols {
			if v, ok := entry.Data[col]; ok {
				parts = append(parts, fmt.Sprintf("%s=%v", col, v))
			}
		}
		var tagParts []string
		for _, col := range breakdownCols {
			if v, ok := entry.Data[col]; ok {
				tagParts = append(tagParts, fmt.Sprintf("%v", v))
			}
		}

		texts = append(texts, strings.Join(parts, ", "))
		urls = append(urls, opts.QueryURL)
		tags = append(tags, strings.Join(tagParts, ","))
	}

	frame := data.NewFrame("annotations",
		data.NewField("time", nil, times),
		data.NewField("text", nil, texts),
		data.NewField("url", nil, urls),
		data.NewField("tags", nil, tags),
	)
	frame.Meta = &data.FrameMeta{Custom: map[string]interface{}{"isAnnotations": true}}
	return data.Frames{frame}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// calcColumnNames derives the display names for calculation columns from the
// query's calculation specs, matching how Honeycomb names them in results.
func calcColumnNames(calcs []honeycomb.Calculation) []string {
	seen := make(map[string]bool)
	var cols []string
	for _, c := range calcs {
		var name string
		if c.Alias != "" {
			name = c.Alias
		} else if c.Column != "" {
			name = fmt.Sprintf("%s(%s)", c.Op, c.Column)
		} else {
			name = c.Op
		}
		if !seen[name] {
			seen[name] = true
			cols = append(cols, name)
		}
	}
	return cols
}

// extractBreakdowns builds a Labels map from the breakdown columns in a series row.
func extractBreakdowns(row map[string]interface{}, breakdowns []string) data.Labels {
	if len(breakdowns) == 0 {
		return nil
	}
	labels := make(data.Labels, len(breakdowns))
	for _, col := range breakdowns {
		if v, ok := row[col]; ok && v != nil {
			labels[col] = fmt.Sprintf("%v", v)
		}
	}
	return labels
}

// groupKey produces a deterministic string key for a set of labels.
func groupKey(labels data.Labels) string {
	if len(labels) == 0 {
		return "__default__"
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(labels[k])
	}
	return sb.String()
}

// labeledName produces a display name combining the column name and group labels.
func labeledName(col string, labels data.Labels) string {
	if len(labels) == 0 {
		return col
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString(col)
	sb.WriteString(" {")
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(labels[k])
	}
	sb.WriteByte('}')
	return sb.String()
}

// toFloat64 converts a JSON-decoded interface{} value to float64.
// JSON numbers are decoded as float64 by default; this handles the
// json.Number case too.
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint64:
		return float64(t)
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
	}
	return 0
}

// emptyFrame returns a minimal valid frame to signal "no data" to Grafana.
func emptyFrame(opts FrameOptions) *data.Frame {
	frame := data.NewFrame("honeycomb")
	SetFrameMeta(frame, opts.QueryURL, opts.GraphURL)
	return frame
}
