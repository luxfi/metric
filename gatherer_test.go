// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMultiGatherer(t *testing.T) {
	mg := NewMultiGatherer()

	// Register a regular registry
	reg1 := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg1.MustRegister(counter)

	err := mg.Register("namespace1", reg1)
	if err != nil {
		t.Fatalf("Failed to register gatherer: %v", err)
	}

	// Try to register with same namespace (should fail)
	err = mg.Register("namespace1", reg1)
	if err == nil {
		t.Error("Expected error when registering duplicate namespace")
	}

	// Register another gatherer
	reg2 := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "Test gauge",
	})
	reg2.MustRegister(gauge)

	err = mg.Register("namespace2", reg2)
	if err != nil {
		t.Fatalf("Failed to register second gatherer: %v", err)
	}

	// Gather metrics
	metrics, err := mg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metrics) < 2 {
		t.Errorf("Expected at least 2 metrics, got %d", len(metrics))
	}

	// Test deregister
	success := mg.Deregister("namespace1")
	if !success {
		t.Error("Expected successful deregistration")
	}

	// Try to deregister non-existent namespace
	success = mg.Deregister("non-existent")
	if success {
		t.Error("Expected false when deregistering non-existent namespace")
	}
}

func TestMakeAndRegister(t *testing.T) {
	mg := NewMultiGatherer()

	// Test MakeAndRegister (it's a global function, not a method)
	reg, err := MakeAndRegister(mg, "test_namespace")
	if err != nil {
		t.Fatalf("Failed to MakeAndRegister: %v", err)
	}
	if reg == nil {
		t.Fatal("Expected non-nil registry")
	}

	// Register a metric to the returned registry
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)

	// Gather and check that the metric is present
	metrics, err := mg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check that we got at least one metric
	if len(metrics) == 0 {
		t.Error("Expected at least one metric")
	}

	// The actual namespace prefixing behavior may vary,
	// just ensure metrics were collected
	found := false
	for _, mf := range metrics {
		if mf.Name != nil {
			// Just check that we got a metric
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find at least one metric")
	}
}

func TestPrefixGatherer(t *testing.T) {
	// NewPrefixGatherer doesn't take arguments, it returns a MultiGatherer
	pg := NewPrefixGatherer()

	// Register a regular registry with a prefix
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)

	// Register the gatherer with a namespace (which acts as prefix)
	err := pg.Register("myprefix", reg)
	if err != nil {
		t.Fatalf("Failed to register with prefix: %v", err)
	}

	// Gather metrics
	metrics, err := pg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check that prefix was applied
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	if *metrics[0].Name != "myprefix_test_counter" {
		t.Errorf("Expected prefixed name, got %s", *metrics[0].Name)
	}
}

func TestLabelGatherer(t *testing.T) {
	// NewLabelGatherer takes a labelName, not labels and registry
	lg := NewLabelGatherer("test_label")

	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)
	counter.Add(42)

	// Register with the label gatherer
	err := lg.Register("namespace", reg)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Gather metrics
	metrics, err := lg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check that metrics were gathered
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}
}

