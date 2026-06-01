package transform

import (
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestUIBaseURL(t *testing.T) {
	tests := []struct {
		name string
		api  string
		want string
	}{
		{"US production", "https://api.honeycomb.io", "https://ui.honeycomb.io"},
		{"EU production", "https://api.eu1.honeycomb.io", "https://ui.eu1.honeycomb.io"},
		{"trailing slash", "https://api.honeycomb.io/", "https://ui.honeycomb.io"},
		{"empty falls back", "", "https://ui.honeycomb.io"},
		{"no api prefix falls back", "https://example.com", "https://example.com"},
		{"api- prefix variant", "https://api-foo.honeycomb.io", "https://ui-foo.honeycomb.io"},
		{"malformed falls back", "://bad", "https://ui.honeycomb.io"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uiBaseURL(tt.api)
			if got != tt.want {
				t.Errorf("uiBaseURL(%q) = %q, want %q", tt.api, got, tt.want)
			}
		})
	}
}

func TestBuildTraceURLTemplate(t *testing.T) {
	tests := []struct {
		name        string
		uiBase      string
		team        string
		environment string
		dataset     string
		want        string
	}{
		{
			name:        "modern with environment",
			uiBase:      "https://ui.honeycomb.io",
			team:        "paddle",
			environment: "production",
			dataset:     "api-gateway-public",
			want:        "https://ui.honeycomb.io/paddle/environments/production/datasets/api-gateway-public/trace?trace_id=${__value.raw}",
		},
		{
			name:        "classic without environment",
			uiBase:      "https://ui.honeycomb.io",
			team:        "paddle",
			environment: "",
			dataset:     "events",
			want:        "https://ui.honeycomb.io/paddle/datasets/events/trace?trace_id=${__value.raw}",
		},
		{
			name:        "EU host",
			uiBase:      "https://ui.eu1.honeycomb.io",
			team:        "team-eu",
			environment: "prod",
			dataset:     "svc",
			want:        "https://ui.eu1.honeycomb.io/team-eu/environments/prod/datasets/svc/trace?trace_id=${__value.raw}",
		},
		{
			name:        "team and dataset escaped",
			uiBase:      "https://ui.honeycomb.io",
			team:        "team with spaces",
			environment: "",
			dataset:     "data set",
			want:        "https://ui.honeycomb.io/team%20with%20spaces/datasets/data%20set/trace?trace_id=${__value.raw}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTraceURLTemplate(tt.uiBase, tt.team, tt.environment, tt.dataset)
			if got != tt.want {
				t.Errorf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestAttachTraceLinks_AttachesToTraceIDColumns(t *testing.T) {
	frame := data.NewFrame(
		"test",
		data.NewField("trace.trace_id", nil, []string{"abc"}),
		data.NewField("name", nil, []string{"GET /foo"}),
		data.NewField("trace_id", nil, []string{"def"}),
		data.NewField("count", nil, []*float64{ptr(1.0)}),
	)

	AttachTraceLinks(frame, "https://api.honeycomb.io", "paddle", "production", "api-gateway-public")

	for _, f := range frame.Fields {
		isTrace := f.Name == "trace.trace_id" || f.Name == "trace_id"
		hasLinks := f.Config != nil && len(f.Config.Links) > 0
		if isTrace && !hasLinks {
			t.Errorf("expected trace link on field %q, got none", f.Name)
		}
		if !isTrace && hasLinks {
			t.Errorf("expected NO trace link on field %q, got %d", f.Name, len(f.Config.Links))
		}
		if isTrace && hasLinks {
			link := f.Config.Links[0]
			if link.Title != "Open trace in Honeycomb" {
				t.Errorf("unexpected link title: %q", link.Title)
			}
			if !link.TargetBlank {
				t.Error("expected TargetBlank=true")
			}
			if !strings.Contains(link.URL, "/paddle/environments/production/datasets/api-gateway-public/trace") {
				t.Errorf("URL missing expected path: %s", link.URL)
			}
			if !strings.Contains(link.URL, "trace_id=${__value.raw}") {
				t.Errorf("URL missing trace_id placeholder: %s", link.URL)
			}
		}
	}
}

func TestAttachTraceLinks_NoOpsWhenTeamMissing(t *testing.T) {
	frame := data.NewFrame("test",
		data.NewField("trace.trace_id", nil, []string{"abc"}),
	)
	AttachTraceLinks(frame, "https://api.honeycomb.io", "", "production", "ds")
	if frame.Fields[0].Config != nil && len(frame.Fields[0].Config.Links) > 0 {
		t.Error("expected no links when team is empty")
	}
}

func TestAttachTraceLinks_NoOpsWhenDatasetMissing(t *testing.T) {
	frame := data.NewFrame("test",
		data.NewField("trace.trace_id", nil, []string{"abc"}),
	)
	AttachTraceLinks(frame, "https://api.honeycomb.io", "paddle", "production", "")
	if frame.Fields[0].Config != nil && len(frame.Fields[0].Config.Links) > 0 {
		t.Error("expected no links when dataset is empty")
	}
}

func TestAttachTraceLinks_NoOpsOnNilFrame(t *testing.T) {
	// Should not panic.
	AttachTraceLinks(nil, "https://api.honeycomb.io", "team", "env", "ds")
}

func TestAttachTraceLinks_ClassicAccountOmitsEnvironment(t *testing.T) {
	frame := data.NewFrame("test",
		data.NewField("trace.trace_id", nil, []string{"abc"}),
	)
	AttachTraceLinks(frame, "https://api.honeycomb.io", "classic-team", "", "events")
	if len(frame.Fields[0].Config.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(frame.Fields[0].Config.Links))
	}
	url := frame.Fields[0].Config.Links[0].URL
	if strings.Contains(url, "/environments/") {
		t.Errorf("classic URL should not contain /environments/: %s", url)
	}
	if !strings.Contains(url, "/classic-team/datasets/events/trace") {
		t.Errorf("classic URL missing expected path: %s", url)
	}
}

func ptr[T any](v T) *T { return &v }
