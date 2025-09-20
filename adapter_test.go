// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestToPrometheusGatherer(t *testing.T) {
	// Test with standard prometheus registry
	promReg := prometheus.NewRegistry()
	gatherer := ToPrometheusGatherer(promReg)

	if gatherer == nil {
		t.Fatal("Expected non-nil gatherer")
	}

	// Test gathering with empty registry
	mfs, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mfs) != 0 {
		t.Errorf("Expected empty metrics, got %d", len(mfs))
	}
}

func TestToPrometheusRegisterer(t *testing.T) {
	// Test with standard prometheus registry
	promReg := prometheus.NewRegistry()
	registerer := ToPrometheusRegisterer(promReg)

	if registerer == nil {
		t.Fatal("Expected non-nil registerer")
	}

	// Test registering a metric
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	err := registerer.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}
}

func TestWrapPrometheusRegistererWith(t *testing.T) {
	promReg := prometheus.NewRegistry()

	// Wrap with labels
	labels := prometheus.Labels{
		"env":     "test",
		"service": "metric",
	}

	wrapped := WrapPrometheusRegistererWith(labels, promReg)

	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped registerer")
	}

	// Register a counter through wrapped registerer
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	err := wrapped.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}
}

func TestWrapPrometheusRegistererWithPrefix(t *testing.T) {
	promReg := prometheus.NewRegistry()

	// Wrap with prefix
	wrapped := WrapPrometheusRegistererWithPrefix("myapp", promReg)

	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped registerer")
	}

	// Register a counter through wrapped registerer
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	err := wrapped.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}
}

func TestNewPrometheusDesc(t *testing.T) {
	// Create a descriptor
	desc := NewPrometheusDesc(
		"test_metric",
		"Test metric description",
		[]string{"label1", "label2"},
		prometheus.Labels{"const": "value"},
	)

	if desc == nil {
		t.Fatal("Expected non-nil descriptor")
	}

	// Test descriptor string representation
	descStr := desc.String()
	if descStr == "" {
		t.Error("Expected non-empty descriptor string")
	}
}

func TestMustNewPrometheusConstMetric(t *testing.T) {
	desc := prometheus.NewDesc(
		"test_metric",
		"Test metric",
		nil,
		nil,
	)

	// Test creating a const metric
	metric := MustNewPrometheusConstMetric(
		desc,
		prometheus.GaugeValue,
		42.0,
	)

	if metric == nil {
		t.Fatal("Expected non-nil metric")
	}

	// Test with labels
	descWithLabels := prometheus.NewDesc(
		"test_metric_labels",
		"Test metric with labels",
		[]string{"label1", "label2"},
		nil,
	)

	metricWithLabels := MustNewPrometheusConstMetric(
		descWithLabels,
		prometheus.CounterValue,
		100.0,
		"value1", "value2",
	)

	if metricWithLabels == nil {
		t.Fatal("Expected non-nil metric with labels")
	}
}