package plugin

import (
	"testing"
	"time"
)

func TestClampFrom(t *testing.T) {
	to := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		from       time.Time
		windowDays int
		want       time.Time
	}{
		{
			name:       "zero window disables clamping",
			from:       to.Add(-90 * 24 * time.Hour),
			windowDays: 0,
			want:       to.Add(-90 * 24 * time.Hour),
		},
		{
			name:       "negative window disables clamping",
			from:       to.Add(-90 * 24 * time.Hour),
			windowDays: -1,
			want:       to.Add(-90 * 24 * time.Hour),
		},
		{
			name:       "from inside window unchanged",
			from:       to.Add(-3 * 24 * time.Hour),
			windowDays: 7,
			want:       to.Add(-3 * 24 * time.Hour),
		},
		{
			name:       "from at exact window boundary unchanged",
			from:       to.Add(-7 * 24 * time.Hour),
			windowDays: 7,
			want:       to.Add(-7 * 24 * time.Hour),
		},
		{
			name:       "from outside window clamped",
			from:       to.Add(-30 * 24 * time.Hour),
			windowDays: 7,
			want:       to.Add(-7 * 24 * time.Hour),
		},
		{
			name:       "from after to (degenerate) unchanged",
			from:       to.Add(time.Hour),
			windowDays: 7,
			want:       to.Add(time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampFrom(tt.from, to, tt.windowDays)
			if !got.Equal(tt.want) {
				t.Errorf("clampFrom(%v, %v, %d) = %v, want %v",
					tt.from, to, tt.windowDays, got, tt.want)
			}
		})
	}
}
