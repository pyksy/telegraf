package testutil

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
)

func TestRequireMetricEqual(t *testing.T) {
	tests := []struct {
		name string
		got  telegraf.Metric
		want telegraf.Metric
	}{
		{
			name: "equal metrics should be equal",
			got: func() telegraf.Metric {
				m := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": 1,
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
			want: func() telegraf.Metric {
				m := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": int64(1),
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricEqual(t, tt.want, tt.got)
		})
	}
}

func TestRequireMetricsEqual(t *testing.T) {
	tests := []struct {
		name string
		got  []telegraf.Metric
		want []telegraf.Metric
		opts []cmp.Option
	}{
		{
			name: "sort metrics option sorts by name",
			got: []telegraf.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			want: []telegraf.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsEqual(t, tt.want, tt.got, tt.opts...)
		})
	}
}

func TestRequireMetricsSubset(t *testing.T) {
	tests := []struct {
		name string
		got  []telegraf.Metric
		want []telegraf.Metric
		opts []cmp.Option
	}{
		{
			name: "subset of metrics",
			got: []telegraf.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(3.14)},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(42)},
					time.Unix(0, 0),
				),
				MustMetric(
					"superfluous",
					map[string]string{},
					map[string]interface{}{"value": true},
					time.Unix(0, 0),
				),
			},
			want: []telegraf.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(42)},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(3.14)},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsSubset(t, tt.want, tt.got, tt.opts...)
		})
	}
}

func TestRequireMetricsStructureEqual(t *testing.T) {
	tests := []struct {
		name string
		got  []telegraf.Metric
		want []telegraf.Metric
		opts []cmp.Option
	}{
		{
			name: "compare structure",
			got: []telegraf.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(3.14)},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(42)},
					time.Unix(0, 0),
				),
			},
			want: []telegraf.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(0)},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(0)},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsStructureEqual(t, tt.want, tt.got, tt.opts...)
		})
	}
}

func TestRequireMetricsStructureSubset(t *testing.T) {
	tests := []struct {
		name string
		got  []telegraf.Metric
		want []telegraf.Metric
		opts []cmp.Option
	}{
		{
			name: "subset of metric structure",
			got: []telegraf.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(3.14)},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(42)},
					time.Unix(0, 0),
				),
				MustMetric(
					"superfluous",
					map[string]string{},
					map[string]interface{}{"value": true},
					time.Unix(0, 0),
				),
			},
			want: []telegraf.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{"value": int64(0)},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{"value": float64(0)},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsStructureSubset(t, tt.want, tt.got, tt.opts...)
		})
	}
}
