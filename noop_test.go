// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"sync"
	"testing"
)

// TestNoopCounterConcurrent guards against the 2026-05-31 regression where
// noopCounter.value was a plain float64 mutated by `n.value++`, causing
// -race failures in threshold protocols.
func TestNoopCounterConcurrent(t *testing.T) {
	c := &noopCounter{}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				c.Inc()
				c.Add(1)
				_ = c.Get()
			}
		}()
	}
	wg.Wait()
	if got, want := c.Get(), float64(64*1000*2); got != want {
		t.Fatalf("noopCounter lost increments under contention: got %v want %v", got, want)
	}
}

// TestNoopGaugeConcurrent guards the same regression for noopGauge.
func TestNoopGaugeConcurrent(t *testing.T) {
	g := &noopGauge{}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				g.Inc()
				g.Add(1)
				g.Dec()
				g.Sub(1)
				g.Set(0)
				_ = g.Get()
			}
		}()
	}
	wg.Wait()
	// Net delta per iteration is 0 (Inc+Add-Dec-Sub then Set(0)), so final
	// must be exactly 0 if no updates were lost or torn.
	if got := g.Get(); got != 0 {
		t.Fatalf("noopGauge lost updates under contention: got %v want 0", got)
	}
}
