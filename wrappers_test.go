// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestWrapPrometheusCounter(t *testing.T) {
	promCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	wrapped := WrapPrometheusCounter(promCounter)
	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped counter")
	}

	// Test that it implements Counter interface
	var _ Counter = wrapped

	// Test operations
	wrapped.Inc()
	wrapped.Add(5)
}

func TestWrapPrometheusGauge(t *testing.T) {
	promGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "Test gauge",
	})

	wrapped := WrapPrometheusGauge(promGauge)
	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped gauge")
	}

	// Test that it implements Gauge interface
	var _ Gauge = wrapped

	// Test operations
	wrapped.Set(42)
	wrapped.Inc()
	wrapped.Dec()
	wrapped.Add(5)
	wrapped.Sub(3)
}

func TestWrapPrometheusCounterVec(t *testing.T) {
	promCounterVec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_counter_vec",
			Help: "Test counter vec",
		},
		[]string{"label1", "label2"},
	)

	wrapped := WrapPrometheusCounterVec(promCounterVec)
	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped counter vec")
	}

	// Test that it implements CounterVec interface
	var _ CounterVec = wrapped

	// Test operations
	counter := wrapped.WithLabelValues("val1", "val2")
	counter.Inc()
	counter.Add(5)
}

func TestWrapPrometheusGaugeVec(t *testing.T) {
	promGaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_gauge_vec",
			Help: "Test gauge vec",
		},
		[]string{"label"},
	)

	wrapped := WrapPrometheusGaugeVec(promGaugeVec)
	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped gauge vec")
	}

	// Test that it implements GaugeVec interface
	var _ GaugeVec = wrapped

	// Test operations
	gauge := wrapped.WithLabelValues("value")
	gauge.Set(42)
}

func TestNewCounterWithOpts(t *testing.T) {
	counter := NewCounterWithOpts(prometheus.CounterOpts{
		Name: "test_counter_opts",
		Help: "Test counter with opts",
	})

	if counter == nil {
		t.Fatal("Expected non-nil counter")
	}

	// Test operations
	counter.Inc()
	counter.Add(10)
}

func TestNewGaugeWithOpts(t *testing.T) {
	gauge := NewGaugeWithOpts(prometheus.GaugeOpts{
		Name: "test_gauge_opts",
		Help: "Test gauge with opts",
	})

	if gauge == nil {
		t.Fatal("Expected non-nil gauge")
	}

	// Test operations
	gauge.Set(100)
	gauge.Inc()
}

func TestNewCounterVecWithOpts(t *testing.T) {
	counterVec := NewCounterVecWithOpts(prometheus.CounterOpts{
		Name: "test_counter_vec_opts",
		Help: "Test counter vec with opts",
	}, []string{"label"})

	if counterVec == nil {
		t.Fatal("Expected non-nil counter vec")
	}

	// Test operations
	counter := counterVec.WithLabelValues("value")
	counter.Inc()
}

func TestNewGaugeVecWithOpts(t *testing.T) {
	gaugeVec := NewGaugeVecWithOpts(prometheus.GaugeOpts{
		Name: "test_gauge_vec_opts",
		Help: "Test gauge vec with opts",
	}, []string{"label"})

	if gaugeVec == nil {
		t.Fatal("Expected non-nil gauge vec")
	}

	// Test operations
	gauge := gaugeVec.WithLabelValues("value")
	gauge.Set(200)
}

func TestAsCollector(t *testing.T) {
	// Test with Counter
	counter := &prometheusCounter{
		counter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "Test counter",
		}),
	}
	counterCollector := AsCollector(counter)
	if counterCollector == nil {
		t.Error("Expected non-nil collector for counter")
	}

	// Test with Gauge
	gauge := &prometheusGauge{
		gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_gauge",
			Help: "Test gauge",
		}),
	}
	gaugeCollector := AsCollector(gauge)
	if gaugeCollector == nil {
		t.Error("Expected non-nil collector for gauge")
	}

	// Test with unknown type
	unknownCollector := AsCollector("unknown")
	if unknownCollector != nil {
		t.Error("Expected nil collector for unknown type")
	}
}

