//go:build metrics

package metric

import "testing"

func TestGaugeBasic(t *testing.T) {
	reg := NewRegistry()
	g := reg.NewGauge("gauge_value", "gauge help")
	g.Set(1.25)
	g.Add(2.0)
	g.Dec()
	g.Inc()

	families := gatherFamilies(t, reg)
	f := findFamily(t, families, "gauge_value")
	if f.Type != MetricTypeGauge {
		t.Fatalf("expected gauge type, got %v", f.Type)
	}
	if len(f.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(f.Metrics))
	}
	got := f.Metrics[0].Value.Value
	if got != 3.25 {
		t.Fatalf("unexpected gauge value: got %v, want 3.25", got)
	}
}

func TestGaugeVec(t *testing.T) {
	reg := NewRegistry()
	gv := reg.NewGaugeVec("inflight", "inflight", []string{"queue"})
	gv.WithLabelValues("a").Set(5)
	gv.WithLabelValues("b").Add(1)

	families := gatherFamilies(t, reg)
	f := findFamily(t, families, "inflight")
	if len(f.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(f.Metrics))
	}
	if m, ok := findMetricWithLabels(f, Labels{"queue": "a"}); !ok || m.Value.Value != 5 {
		t.Fatalf("missing queue a metric")
	}
	if m, ok := findMetricWithLabels(f, Labels{"queue": "b"}); !ok || m.Value.Value != 1 {
		t.Fatalf("missing queue b metric")
	}
}
