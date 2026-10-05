package ginprometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewMetric(t *testing.T) {
	tests := []struct {
		want   any
		metric *Metric
		name   string
	}{
		{name: "nil metric", metric: nil, want: nil},
		{name: "counter vec", metric: &Metric{Name: "m", Type: MetricTypeCounterVec, Args: []string{"a"}}, want: &prometheus.CounterVec{}},
		{name: "counter", metric: &Metric{Name: "m", Type: MetricTypeCounter}, want: (*prometheus.Counter)(nil)},
		{name: "gauge vec", metric: &Metric{Name: "m", Type: MetricTypeGaugeVec, Args: []string{"a"}}, want: &prometheus.GaugeVec{}},
		{name: "gauge", metric: &Metric{Name: "m", Type: MetricTypeGauge}, want: (*prometheus.Gauge)(nil)},
		{name: "histogram vec", metric: &Metric{Name: "m", Type: MetricTypeHistogramVec, Args: []string{"a"}}, want: &prometheus.HistogramVec{}},
		{name: "histogram", metric: &Metric{Name: "m", Type: MetricTypeHistogram}, want: (*prometheus.Histogram)(nil)},
		{name: "summary vec", metric: &Metric{Name: "m", Type: MetricTypeSummaryVec, Args: []string{"a"}}, want: &prometheus.SummaryVec{}},
		{name: "summary", metric: &Metric{Name: "m", Type: MetricTypeSummary}, want: (*prometheus.Summary)(nil)},
		{name: "unknown type", metric: &Metric{Name: "m", Type: MetricType(-1)}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetric(tt.metric, "test")
			switch want := tt.want.(type) {
			case nil:
				assert.Nil(t, got)
			case *prometheus.Counter:
				assert.Implements(t, want, got)
			case *prometheus.Gauge:
				assert.Implements(t, want, got)
			case *prometheus.Histogram:
				assert.Implements(t, want, got)
			case *prometheus.Summary:
				assert.Implements(t, want, got)
			default:
				assert.IsType(t, want, got)
			}
		})
	}
}
