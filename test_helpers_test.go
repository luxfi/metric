//go:build !metrics_noop

package metric

import "testing"

func gatherFamilies(t *testing.T, reg Registry) []*MetricFamily {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	return families
}

func findFamily(t *testing.T, families []*MetricFamily, name string) *MetricFamily {
	t.Helper()
	for _, f := range families {
		if f != nil && f.Name == name {
			return f
		}
	}
	t.Fatalf("missing family %q", name)
	return nil
}

func labelsMatch(pairs []LabelPair, want Labels) bool {
	if len(want) == 0 {
		return len(pairs) == 0
	}
	if len(pairs) != len(want) {
		return false
	}
	for _, p := range pairs {
		if want[p.Name] != p.Value {
			return false
		}
	}
	return true
}

func findMetricWithLabels(f *MetricFamily, labels Labels) (Metric, bool) {
	for _, m := range f.Metrics {
		if labelsMatch(m.Labels, labels) {
			return m, true
		}
	}
	return Metric{}, false
}
