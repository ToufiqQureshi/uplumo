package sitereport

import (
	"regexp"
	"slices"
	"testing"
)

var axisLabel = regexp.MustCompile(`<text class="axis" x="40\.0" y="[0-9.]+" text-anchor="end">([^<]+)</text>`)

func TestLineChartAxisLabelsAreDistinctWholeSteps(t *testing.T) {
	tests := []struct {
		name string
		max  float64
		want []string
	}{
		{"two clicks", 2, []string{"4", "3", "2", "1", "0"}},
		{"three visits", 3, []string{"4", "3", "2", "1", "0"}},
		{"zero", 0, []string{"4", "3", "2", "1", "0"}},
		{"twenty impressions", 20, []string{"20", "15", "10", "5", "0"}},
		{"twenty five", 25, []string{"40", "30", "20", "10", "0"}},
		{"thousands", 3_400, []string{"4,000", "3,000", "2,000", "1,000", "0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := string(lineChart("c", []float64{0, tt.max}, []string{"2026-10-01", "2026-10-02"}, "clicks"))
			got := []string{}
			for _, m := range axisLabel.FindAllStringSubmatch(svg, -1) {
				got = append(got, m[1])
			}
			slices.Reverse(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("labels = %v, want %v", got, tt.want)
			}
		})
	}
}
