// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusCounterGet(t *testing.T) {
	counter := &prometheusCounter{
		counter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "Test counter",
		}),
	}

	counter.Add(42)

	// Get is not supported for Prometheus counters, should return 0
	value := counter.Get()
	if value != 0 {
		t.Errorf("Expected Get() to return 0 for prometheus counter, got %f", value)
	}
}

func TestPrometheusGaugeGet(t *testing.T) {
	gauge := &prometheusGauge{
		gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_gauge",
			Help: "Test gauge",
		}),
	}

	gauge.Set(42)

	// Get is not supported for Prometheus gauges, should return 0
	value := gauge.Get()
	if value != 0 {
		t.Errorf("Expected Get() to return 0 for prometheus gauge, got %f", value)
	}
}

func TestPrometheusDescribeCollect(t *testing.T) {
	// Test Counter
	counter := &prometheusCounter{
		counter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "Test counter",
		}),
	}

	descCh := make(chan *prometheus.Desc, 1)
	counter.Describe(descCh)
	close(descCh)

	metricCh := make(chan prometheus.Metric, 1)
	counter.Collect(metricCh)
	close(metricCh)

	// Test Gauge
	gauge := &prometheusGauge{
		gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_gauge",
			Help: "Test gauge",
		}),
	}

	descCh2 := make(chan *prometheus.Desc, 1)
	gauge.Describe(descCh2)
	close(descCh2)

	metricCh2 := make(chan prometheus.Metric, 1)
	gauge.Collect(metricCh2)
	close(metricCh2)

	// Test Histogram
	histogram := &prometheusHistogram{
		histogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "test_histogram",
			Help: "Test histogram",
		}),
	}

	descCh3 := make(chan *prometheus.Desc, 1)
	histogram.Describe(descCh3)
	close(descCh3)

	metricCh3 := make(chan prometheus.Metric, 1)
	histogram.Collect(metricCh3)
	close(metricCh3)

	// Test Summary
	summary := &prometheusSummary{
		summary: prometheus.NewSummary(prometheus.SummaryOpts{
			Name: "test_summary",
			Help: "Test summary",
		}),
	}

	descCh4 := make(chan *prometheus.Desc, 1)
	summary.Describe(descCh4)
	close(descCh4)

	metricCh4 := make(chan prometheus.Metric, 1)
	summary.Collect(metricCh4)
	close(metricCh4)
}

func TestPrometheusVectorDescribeCollect(t *testing.T) {
	// Test CounterVec
	counterVec := &prometheusCounterVec{
		vec: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_counter_vec",
				Help: "Test counter vec",
			},
			[]string{"label"},
		),
	}

	descCh := make(chan *prometheus.Desc, 1)
	counterVec.Describe(descCh)
	close(descCh)

	metricCh := make(chan prometheus.Metric, 1)
	counterVec.Collect(metricCh)
	close(metricCh)

	// Test GaugeVec
	gaugeVec := &prometheusGaugeVec{
		vec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "test_gauge_vec",
				Help: "Test gauge vec",
			},
			[]string{"label"},
		),
	}

	descCh2 := make(chan *prometheus.Desc, 1)
	gaugeVec.Describe(descCh2)
	close(descCh2)

	metricCh2 := make(chan prometheus.Metric, 1)
	gaugeVec.Collect(metricCh2)
	close(metricCh2)

	// Test HistogramVec - These don't have Describe/Collect methods directly
	histogramVec := &prometheusHistogramVec{
		vec: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "test_histogram_vec",
				Help: "Test histogram vec",
			},
			[]string{"label"},
		),
	}
	// Just ensure it can be created without panicking
	_ = histogramVec

	// Test SummaryVec - These don't have Describe/Collect methods directly
	summaryVec := &prometheusSummaryVec{
		vec: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "test_summary_vec",
				Help: "Test summary vec",
			},
			[]string{"label"},
		),
	}
	// Just ensure it can be created without panicking
	_ = summaryVec
}

func TestPrometheusGaugeVecWith(t *testing.T) {
	gaugeVec := &prometheusGaugeVec{
		vec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "test_gauge_vec",
				Help: "Test gauge vec",
			},
			[]string{"label1", "label2"},
		),
	}

	labels := Labels{"label1": "value1", "label2": "value2"}
	gauge := gaugeVec.With(labels)

	if gauge == nil {
		t.Fatal("Expected non-nil gauge")
	}

	// Set a value
	gauge.Set(42)
}

func TestPrometheusFactoryWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	factory := NewPrometheusFactoryWithRegistry(registry)

	if factory == nil {
		t.Fatal("Expected non-nil factory")
	}

	metrics := factory.New("test")
	counter := metrics.NewCounter("counter", "help")
	counter.Inc()

	// Verify the metric was registered with the provided registry
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if *mf.Name == "test_counter" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find test_counter in registry")
	}
}

func TestPrometheusFactoryNewWithRegistry(t *testing.T) {
	factory := NewPrometheusFactory()
	registry := prometheus.NewRegistry()

	metrics := factory.NewWithRegistry("test", registry)
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	counter := metrics.NewCounter("counter", "help")
	counter.Inc()

	// Verify the metric was registered with the provided registry
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if *mf.Name == "test_counter" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find test_counter in registry")
	}
}

func TestPrometheusMetricsRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := &prometheusMetrics{
		registry:  registry,
		namespace: "test",
	}

	// Test Registry method
	reg := metrics.Registry()
	if reg != registry {
		t.Error("Expected Registry() to return the same registry")
	}

	// Test PrometheusRegistry method
	promReg := metrics.PrometheusRegistry()
	if promReg != registry {
		t.Error("Expected PrometheusRegistry() to return the same registry")
	}
}

func TestNewPrometheusMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetrics("test", registry)
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Test that it creates metrics properly
	counter := metrics.NewCounter("counter", "help")
	counter.Inc()

	gauge := metrics.NewGauge("gauge", "help")
	gauge.Set(42)
}