//go:build metrics

// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import "testing"

// TestMustCurryWith verifies that currying a subset of labels and then
// supplying the rest addresses the same child as supplying all labels at once,
// across With and WithLabelValues. Mirrors the replicate usage pattern.
func TestMustCurryWith(t *testing.T) {
	reg := NewRegistry()
	vec := reg.NewCounterVec("curry_total", "help", []string{"db", "mode"})

	cur := vec.MustCurryWith(Labels{"db": "foo"})
	cur.With(Labels{"mode": "passive"}).Inc()
	cur.WithLabelValues("passive").Inc() // remaining names == [mode]
	vec.With(Labels{"db": "foo", "mode": "passive"}).Add(3)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var total float64
	var series int
	for _, mf := range families {
		if mf.Name != "curry_total" {
			continue
		}
		for _, m := range mf.Metrics {
			series++
			total += m.Value.Value
		}
	}
	// All three calls must land on the SAME db=foo,mode=passive child.
	if series != 1 {
		t.Fatalf("expected 1 series, got %d", series)
	}
	if total != 5 {
		t.Fatalf("expected child value 5 (1+1+3), got %v", total)
	}
}

// TestUnregister verifies a registered name can be dropped and re-registered.
func TestUnregister(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("scrap_total", "help")
	if err := reg.Register(c); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Re-registering the same name must fail while it is present.
	if err := reg.Register(reg.NewCounter("scrap_total", "help")); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
	if !reg.Unregister(c) {
		t.Fatal("expected Unregister to report the name was present")
	}
	// After Unregister the name is free again.
	if err := reg.Register(reg.NewCounter("scrap_total", "help")); err != nil {
		t.Fatalf("re-register after unregister: %v", err)
	}
}

// TestDefaultRegistererGatherer verifies the package-level handles are wired
// to DefaultRegistry and usable as a Registerer/Gatherer pair.
func TestDefaultRegistererGatherer(t *testing.T) {
	if DefaultRegisterer == nil || DefaultGatherer == nil {
		t.Fatal("default registerer/gatherer must be non-nil")
	}
	h := InstrumentMetricHandler(DefaultRegisterer, NewHTTPHandler(DefaultGatherer, HandlerOpts{}))
	if h == nil {
		t.Fatal("instrumented handler must be non-nil")
	}
}
