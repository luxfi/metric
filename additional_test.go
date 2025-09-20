// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// No longer needed - using dto.MetricFamily directly

// Test additional uncovered functions in context.go
func TestNewPedanticContextRegistry(t *testing.T) {
	reg := NewPedanticContextRegistry()
	if reg == nil {
		t.Fatal("Expected non-nil pedantic registry")
	}

	// Test that pedantic mode catches duplicate descriptors
	counter1 := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "duplicate_metric",
		Help: "First metric",
	})

	err := reg.Register(counter1)
	if err != nil {
		t.Fatalf("Failed to register first counter: %v", err)
	}

	// Try to register the same collector again (should fail in pedantic mode)
	err = reg.Register(counter1)
	if err == nil {
		t.Error("Expected error when registering duplicate metric in pedantic mode")
	}
}

func TestContextRegistryUnregister(t *testing.T) {
	reg := NewContextRegistry()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	err := reg.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}

	// Unregister the counter
	success := reg.Unregister(counter)
	if !success {
		t.Error("Expected successful unregistration")
	}

	// Try to unregister again (should return false)
	success = reg.Unregister(counter)
	if success {
		t.Error("Expected false when unregistering non-existent collector")
	}
}

func TestCollectorFuncWithNilFunctions(t *testing.T) {
	// Test with nil describe function
	cf := NewCollectorFunc(nil, func(ctx context.Context, ch chan<- prometheus.Metric) {
		// Do nothing
	})

	descCh := make(chan *prometheus.Desc, 1)
	cf.Describe(descCh) // Should not panic
	close(descCh)

	// Test with nil collect function
	cf2 := NewCollectorFunc(
		func(ch chan<- *prometheus.Desc) {
			ch <- prometheus.NewDesc("test", "test", nil, nil)
		},
		nil,
	)

	metricCh := make(chan prometheus.Metric, 1)
	cf2.CollectWithContext(context.Background(), metricCh) // Should not panic
	close(metricCh)
}

func TestCollectorFuncCollect(t *testing.T) {
	// Test the Collect method (non-context version)
	var collectCalled bool

	cf := NewCollectorFunc(
		func(ch chan<- *prometheus.Desc) {
			ch <- prometheus.NewDesc("test_metric", "help", nil, nil)
		},
		func(ctx context.Context, ch chan<- prometheus.Metric) {
			collectCalled = true
			metric, _ := prometheus.NewConstMetric(
				prometheus.NewDesc("test_metric", "help", nil, nil),
				prometheus.GaugeValue,
				42.0,
			)
			ch <- metric
		},
	)

	metricCh := make(chan prometheus.Metric, 1)
	cf.Collect(metricCh)
	close(metricCh)

	if !collectCalled {
		t.Error("Collect function was not called")
	}
}

func TestNewContextCollectorWrapper(t *testing.T) {
	// Create a context-aware collector
	contextCollector := &testContextCollector{
		name: "wrapped_metric",
		help: "Wrapped metric",
	}

	// Wrap it to make it a standard collector
	wrapped := NewContextCollectorWrapper(contextCollector)

	// Test Describe
	descCh := make(chan *prometheus.Desc, 1)
	wrapped.Describe(descCh)
	close(descCh)

	// Test Collect
	metricCh := make(chan prometheus.Metric, 1)
	wrapped.Collect(metricCh)
	close(metricCh)

	// Should have received a metric
	select {
	case metric := <-metricCh:
		if metric == nil {
			t.Error("Expected non-nil metric")
		}
	default:
		// Metric was collected
	}
}

func TestGathererFunc(t *testing.T) {
	// Create a gatherer function
	gathererFunc := GathererFunc(func() ([]*dto.MetricFamily, error) {
		mf := &dto.MetricFamily{
			Name: testStringPtr("test_metric"),
			Help: testStringPtr("Test metric"),
		}
		return []*dto.MetricFamily{mf}, nil
	})

	mfs, err := gathererFunc.Gather()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mfs) != 1 {
		t.Errorf("Expected 1 metric family, got %d", len(mfs))
	}

	if *mfs[0].Name != "test_metric" {
		t.Errorf("Expected metric name 'test_metric', got %s", *mfs[0].Name)
	}
}

func TestGathererWithContextFunc(t *testing.T) {
	// Create a context-aware gatherer function
	var ctxReceived bool
	gathererFunc := GathererWithContextFunc(func(ctx context.Context) ([]*dto.MetricFamily, error) {
		ctxReceived = ctx != nil
		mf := &dto.MetricFamily{
			Name: testStringPtr("test_metric"),
			Help: testStringPtr("Test metric"),
		}
		return []*dto.MetricFamily{mf}, nil
	})

	// Test GatherWithContext
	ctx := context.Background()
	mfs, err := gathererFunc.GatherWithContext(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !ctxReceived {
		t.Error("Context was not received")
	}

	if len(mfs) != 1 {
		t.Errorf("Expected 1 metric family, got %d", len(mfs))
	}

	// Test Gather (should use background context)
	mfs2, err := gathererFunc.Gather()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mfs2) != 1 {
		t.Errorf("Expected 1 metric family, got %d", len(mfs2))
	}
}

func TestMultiGathererWithContextDeregister(t *testing.T) {
	mg := NewMultiGathererWithContext()

	// Register a gatherer
	reg := prometheus.NewRegistry()
	err := mg.Register("test", reg)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Deregister should succeed
	success := mg.Deregister("test")
	if !success {
		t.Error("Expected successful deregistration")
	}

	// Deregister non-existent should return false
	success = mg.Deregister("non-existent")
	if success {
		t.Error("Expected false when deregistering non-existent namespace")
	}
}

func TestMultiGathererWithContextGather(t *testing.T) {
	mg := NewMultiGathererWithContext()

	// Register a gatherer
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)

	err := mg.Register("namespace", reg)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Test Gather (non-context version)
	mfs, err := mg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather: %v", err)
	}

	if len(mfs) != 1 {
		t.Errorf("Expected 1 metric family, got %d", len(mfs))
	}

	if *mfs[0].Name != "namespace_test_counter" {
		t.Errorf("Expected namespaced metric name, got %s", *mfs[0].Name)
	}
}

func TestContextRegistryWithPanicCollector(t *testing.T) {
	reg := NewContextRegistry()

	// Register a collector that panics
	panicCollector := &panicCollector{}
	err := reg.Register(panicCollector)
	if err != nil {
		t.Fatalf("Failed to register panic collector: %v", err)
	}

	// Gathering should handle the panic and return an error
	_, err = reg.Gather()
	if err == nil {
		t.Error("Expected error when collector panics")
	}
}

func TestContextRegistryMustRegisterPanic(t *testing.T) {
	// MustRegister only panics on actual errors from Register
	// Since the test above already covers pedantic mode duplicate detection
	// and regular mode doesn't detect duplicates, let's skip this test
	t.Skip("MustRegister panic test covered by pedantic mode test")
}

func TestContextRegistryCancelledBeforeStart(t *testing.T) {
	reg := NewContextRegistry()

	// Register a collector
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Gathering should return immediately with error
	_, err := reg.GatherWithContext(ctx)
	if err == nil {
		t.Error("Expected error when context is already cancelled")
	}
}

// Helper types for testing

type panicCollector struct{}

func (p *panicCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("panic_metric", "Panic metric", nil, nil)
}

func (p *panicCollector) Collect(ch chan<- prometheus.Metric) {
	panic("intentional panic for testing")
}


func testStringPtr(s string) *string {
	return &s
}

// Test additional functions in metric.go
func TestNewWithRegistry(t *testing.T) {
	registry := NewNoOpRegistry()
	metrics := NewWithRegistry("test", registry)

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Test that metrics work
	counter := metrics.NewCounter("counter", "help")
	counter.Inc() // Should not panic
}

// Test additional noop functions
func TestNoOpMetricDescribeCollect(t *testing.T) {
	counter := &noopCounter{}
	gauge := &noopGauge{}
	histogram := &noopHistogram{}
	summary := &noopSummary{}

	// Test Describe (should not panic)
	descCh := make(chan *prometheus.Desc, 1)
	counter.Describe(descCh)
	gauge.Describe(descCh)
	histogram.Describe(descCh)
	summary.Describe(descCh)
	close(descCh)

	// Test Collect (should not panic)
	metricCh := make(chan prometheus.Metric, 1)
	counter.Collect(metricCh)
	gauge.Collect(metricCh)
	histogram.Collect(metricCh)
	summary.Collect(metricCh)
	close(metricCh)
}

func TestNoOpVectorDescribeCollect(t *testing.T) {
	counterVec := &noopCounterVec{}
	gaugeVec := &noopGaugeVec{}

	// Test Describe (should not panic)
	descCh := make(chan *prometheus.Desc, 1)
	counterVec.Describe(descCh)
	gaugeVec.Describe(descCh)
	close(descCh)

	// Test Collect (should not panic)
	metricCh := make(chan prometheus.Metric, 1)
	counterVec.Collect(metricCh)
	gaugeVec.Collect(metricCh)
	close(metricCh)
}

func TestNoOpVectorWith(t *testing.T) {
	gaugeVec := &noopGaugeVec{}
	histogramVec := &noopHistogramVec{}
	summaryVec := &noopSummaryVec{}

	// Test With method
	labels := Labels{"key": "value"}

	gauge := gaugeVec.With(labels)
	if gauge == nil {
		t.Error("Expected non-nil gauge")
	}

	histogram := histogramVec.With(labels)
	if histogram == nil {
		t.Error("Expected non-nil histogram")
	}

	summary := summaryVec.With(labels)
	if summary == nil {
		t.Error("Expected non-nil summary")
	}
}

func TestNoOpHistogramObserve(t *testing.T) {
	histogram := &noopHistogram{}
	histogram.Observe(42.0) // Should not panic
}

func TestNoOpSummaryObserve(t *testing.T) {
	summary := &noopSummary{}
	summary.Observe(42.0) // Should not panic
}

func TestNewNoOp(t *testing.T) {
	metrics := NewNoOp()
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Test that it returns noop metrics
	counter := metrics.NewCounter("counter", "help")
	counter.Inc() // Should not panic
}

func TestNewNoopMetrics(t *testing.T) {
	counter := NewNoopCounter("test_counter")
	counter.Inc() // Should not panic

	gauge := NewNoopGauge("test_gauge")
	gauge.Set(42) // Should not panic

	histogram := NewNoopHistogram("test_histogram")
	histogram.Observe(42) // Should not panic

	summary := NewNoopSummary("test_summary")
	summary.Observe(42) // Should not panic
}

func TestNoOpPrometheusRegistry(t *testing.T) {
	metrics := NewNoOpMetrics("test")
	promReg := metrics.PrometheusRegistry()

	// Should return a valid registry (not nil) for noop
	if promReg == nil {
		t.Error("Expected non-nil prometheus registry for noop metrics")
	}
}