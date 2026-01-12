//go:build metrics

package metric

import "testing"

func TestCounterBasic(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("counter_total", "counter help")
	c.Inc()
	c.Add(2.5)

	families := gatherFamilies(t, reg)
	f := findFamily(t, families, "counter_total")
	if f.Type != MetricTypeCounter {
		t.Fatalf("expected counter type, got %v", f.Type)
	}
	if len(f.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(f.Metrics))
	}
	got := f.Metrics[0].Value.Value
	if got != 3.5 {
		t.Fatalf("unexpected counter value: got %v, want 3.5", got)
	}
}

func TestCounterVec(t *testing.T) {
	reg := NewRegistry()
	cv := reg.NewCounterVec("requests_total", "requests", []string{"method", "code"})
	cv.WithLabelValues("GET", "200").Add(2)
	cv.With(Labels{"method": "POST", "code": "500"}).Inc()

	families := gatherFamilies(t, reg)
	f := findFamily(t, families, "requests_total")
	if len(f.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(f.Metrics))
	}
	if m, ok := findMetricWithLabels(f, Labels{"method": "GET", "code": "200"}); !ok || m.Value.Value != 2 {
		t.Fatalf("missing GET/200 metric")
	}
	if m, ok := findMetricWithLabels(f, Labels{"method": "POST", "code": "500"}); !ok || m.Value.Value != 1 {
		t.Fatalf("missing POST/500 metric")
	}
}
