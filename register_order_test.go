//go:build !metrics_noop

// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import "testing"

// A vector is created against one registry and registered with another, and
// the natural order is to create it, use it, then register it. Registration
// repointed the vector for writes still to come; the children it already had
// stayed where they were made, so a counter incremented before registration
// gathered nowhere and the series simply vanished.
func TestRegister_TakesTheChildrenAVectorAlreadyHas(t *testing.T) {
	for _, order := range []string{"use then register", "register then use"} {
		t.Run(order, func(t *testing.T) {
			r := NewRegistry()
			c := NewCounterVec(CounterOpts{Name: "orders_total", Help: "orders"}, []string{"kind"})

			use := func() { c.With(Labels{"kind": "limit"}).Inc() }
			reg := func() {
				if err := r.Register(AsCollector(c)); err != nil {
					t.Fatalf("Register: %v", err)
				}
			}
			if order == "use then register" {
				use()
				reg()
			} else {
				reg()
				use()
			}

			families, err := r.Gather()
			if err != nil {
				t.Fatalf("Gather: %v", err)
			}
			var found *MetricFamily
			for _, f := range families {
				if f.Name == "orders_total" {
					found = f
				}
			}
			if found == nil {
				t.Fatalf("orders_total gathered nowhere; the registry saw %d families", len(families))
			}
			if len(found.Metrics) != 1 {
				t.Fatalf("orders_total has %d series, want 1", len(found.Metrics))
			}
			if got := found.Metrics[0].Value.Value; got != 1 {
				t.Fatalf("orders_total = %v, want 1", got)
			}
		})
	}
}
