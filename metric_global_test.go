// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"testing"
)

func TestGlobalMetricFunctions(t *testing.T) {
	// Test NewCounter
	counter := NewCounter(CounterOpts{
		Name: "test_counter",
		Help: "help text",
	})
	if counter == nil {
		t.Error("Expected non-nil counter")
	}
	counter.Inc()
	counter.Add(5)

	// Test NewCounterVec
	counterVec := NewCounterVec(CounterOpts{
		Name: "test_counter_vec",
		Help: "help text",
	}, []string{"label1", "label2"})
	if counterVec == nil {
		t.Error("Expected non-nil counter vec")
	}
	counterVec.WithLabelValues("val1", "val2").Inc()

	// Test NewGauge
	gauge := NewGauge(GaugeOpts{
		Name: "test_gauge",
		Help: "help text",
	})
	if gauge == nil {
		t.Error("Expected non-nil gauge")
	}
	gauge.Set(42)
	gauge.Inc()
	gauge.Dec()

	// Test NewGaugeVec
	gaugeVec := NewGaugeVec(GaugeOpts{
		Name: "test_gauge_vec",
		Help: "help text",
	}, []string{"label"})
	if gaugeVec == nil {
		t.Error("Expected non-nil gauge vec")
	}
	gaugeVec.WithLabelValues("value").Set(100)
}
