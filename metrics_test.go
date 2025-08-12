// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"testing"
)

func TestNoOpMetrics(t *testing.T) {
	metrics := NewNoOpMetrics("test")
	
	// Test counter
	counter := metrics.NewCounter("test_counter", "Test counter")
	counter.Inc()
	counter.Add(5)
	if counter.Get() != 6 {
		t.Errorf("expected counter value 6, got %f", counter.Get())
	}
	
	// Test gauge
	gauge := metrics.NewGauge("test_gauge", "Test gauge")
	gauge.Set(10)
	gauge.Inc()
	gauge.Dec()
	gauge.Add(5)
	gauge.Sub(3)
	if gauge.Get() != 12 {
		t.Errorf("expected gauge value 12, got %f", gauge.Get())
	}
	
	// Test histogram
	histogram := metrics.NewHistogram("test_histogram", "Test histogram", []float64{1, 5, 10})
	histogram.Observe(3.5) // Should not panic
	
	// Test summary
	summary := metrics.NewSummary("test_summary", "Test summary", map[float64]float64{0.5: 0.05, 0.9: 0.01})
	summary.Observe(100) // Should not panic
	
	// Test vectors
	counterVec := metrics.NewCounterVec("test_counter_vec", "Test counter vec", []string{"label1", "label2"})
	counterVec.WithLabelValues("value1", "value2").Inc()
	counterVec.With(Labels{"label1": "value1", "label2": "value2"}).Add(2)
	
	gaugeVec := metrics.NewGaugeVec("test_gauge_vec", "Test gauge vec", []string{"label"})
	gaugeVec.WithLabelValues("value").Set(42)
	
	histogramVec := metrics.NewHistogramVec("test_histogram_vec", "Test histogram vec", []string{"label"}, []float64{1, 5, 10})
	histogramVec.WithLabelValues("value").Observe(7)
	
	summaryVec := metrics.NewSummaryVec("test_summary_vec", "Test summary vec", []string{"label"}, map[float64]float64{0.5: 0.05})
	summaryVec.WithLabelValues("value").Observe(50)
	
	// Test registry
	registry := metrics.Registry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}
}

func TestNoOpFactory(t *testing.T) {
	factory := NewNoOpFactory()
	
	metrics1 := factory.New("namespace1")
	metrics2 := factory.NewWithRegistry("namespace2", NewNoOpRegistry())
	
	// These should not panic
	metrics1.NewCounter("counter", "help")
	metrics2.NewGauge("gauge", "help")
}

func TestPrometheusMetrics(t *testing.T) {
	// Note: This is a basic test. In production, you'd want more comprehensive tests
	factory := NewPrometheusFactory()
	metrics := factory.New("test")
	
	// Test counter
	counter := metrics.NewCounter("test_counter", "Test counter")
	counter.Inc()
	counter.Add(5)
	
	// Test gauge
	gauge := metrics.NewGauge("test_gauge", "Test gauge")
	gauge.Set(10)
	gauge.Inc()
	gauge.Dec()
	gauge.Add(5)
	gauge.Sub(3)
	
	// Test histogram
	histogram := metrics.NewHistogram("test_histogram", "Test histogram", []float64{1, 5, 10})
	histogram.Observe(3.5)
	
	// Test summary
	summary := metrics.NewSummary("test_summary", "Test summary", map[float64]float64{0.5: 0.05, 0.9: 0.01})
	summary.Observe(100)
	
	// Test vectors
	counterVec := metrics.NewCounterVec("test_counter_vec", "Test counter vec", []string{"label1", "label2"})
	counterVec.WithLabelValues("value1", "value2").Inc()
	counterVec.With(Labels{"label1": "value1", "label2": "value2"}).Add(2)
	
	gaugeVec := metrics.NewGaugeVec("test_gauge_vec", "Test gauge vec", []string{"label"})
	gaugeVec.WithLabelValues("value").Set(42)
	
	histogramVec := metrics.NewHistogramVec("test_histogram_vec", "Test histogram vec", []string{"label"}, []float64{1, 5, 10})
	histogramVec.WithLabelValues("value").Observe(7)
	
	summaryVec := metrics.NewSummaryVec("test_summary_vec", "Test summary vec", []string{"label"}, map[float64]float64{0.5: 0.05})
	summaryVec.WithLabelValues("value").Observe(50)
}

func TestGlobalFactory(t *testing.T) {
	// Test default factory (noop)
	metrics := New("test")
	counter := metrics.NewCounter("counter", "help")
	counter.Inc() // Should not panic
	
	// Test setting a new factory
	SetFactory(NewPrometheusFactory())
	metrics2 := New("test2")
	gauge := metrics2.NewGauge("gauge", "help")
	gauge.Set(42) // Should not panic
	
	// Reset to noop for other tests
	SetFactory(NewNoOpFactory())
}