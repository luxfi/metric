//go:build metrics

package metric

import "testing"

func TestHistogramCounts(t *testing.T) {
	reg := NewRegistry()
	h := reg.NewHistogram("latency_seconds", "latency", []float64{1, 5})
	h.Observe(0.5)
	h.Observe(1.2)
	h.Observe(6.0)

	families := gatherFamilies(t, reg)
	f := findFamily(t, families, "latency_seconds")
	if f.Type != MetricTypeHistogram {
		t.Fatalf("expected histogram type, got %v", f.Type)
	}
	if len(f.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(f.Metrics))
	}
	m := f.Metrics[0]
	if m.Value.SampleCount != 3 {
		t.Fatalf("unexpected sample count %d", m.Value.SampleCount)
	}
	if diff := m.Value.SampleSum - 7.7; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected sample sum %v", m.Value.SampleSum)
	}
	if len(m.Value.Buckets) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(m.Value.Buckets))
	}
	if m.Value.Buckets[0].CumulativeCount != 1 {
		t.Fatalf("bucket <=1 count mismatch")
	}
	if m.Value.Buckets[1].CumulativeCount != 2 {
		t.Fatalf("bucket <=5 count mismatch")
	}
	if m.Value.Buckets[2].CumulativeCount != 3 {
		t.Fatalf("bucket +Inf count mismatch")
	}
}
