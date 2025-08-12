// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TestContextRegistry tests the context-aware registry
func TestContextRegistry(t *testing.T) {
	reg := NewContextRegistry()

	// Test registering a standard collector
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "A test counter",
	})

	err := reg.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}

	// Test gathering metrics
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(mfs) == 0 {
		t.Error("Expected at least one metric family")
	}
}

// TestContextRegistryWithContext tests context propagation through the registry
func TestContextRegistryWithContext(t *testing.T) {
	reg := NewContextRegistry()

	// Create a context-aware collector
	contextCollector := &testContextCollector{
		name: "test_context_metric",
		help: "A test context metric",
	}

	err := reg.Register(contextCollector)
	if err != nil {
		t.Fatalf("Failed to register context collector: %v", err)
	}

	// Test with a normal context
	ctx := context.Background()
	mfs, err := reg.GatherWithContext(ctx)
	if err != nil {
		t.Fatalf("Failed to gather with context: %v", err)
	}

	if len(mfs) == 0 {
		t.Error("Expected at least one metric family")
	}

	// Verify the collector received the context
	if !contextCollector.contextReceived {
		t.Error("Context was not propagated to the collector")
	}
}

// TestContextRegistryTimeout tests timeout handling
func TestContextRegistryTimeout(t *testing.T) {
	reg := NewContextRegistry()

	// Create a slow collector
	slowCollector := &testSlowCollector{
		delay: 2 * time.Second,
		name:  "slow_metric",
	}

	err := reg.Register(slowCollector)
	if err != nil {
		t.Fatalf("Failed to register slow collector: %v", err)
	}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Gathering should timeout
	_, err = reg.GatherWithContext(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded error, got: %v", err)
	}
}

// TestContextRegistryCancellation tests cancellation handling
func TestContextRegistryCancellation(t *testing.T) {
	reg := NewContextRegistry()

	// Create a collector that respects cancellation
	cancelCollector := &testCancellableCollector{
		name: "cancellable_metric",
	}

	err := reg.Register(cancelCollector)
	if err != nil {
		t.Fatalf("Failed to register cancellable collector: %v", err)
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to ensure cancellation happens during collection
	cancel()

	// Try to gather with cancelled context
	_, gatherErr := reg.GatherWithContext(ctx)

	// Should have received a cancellation error
	if gatherErr == nil {
		t.Error("Expected cancellation error")
	}

	if !errors.Is(gatherErr, context.Canceled) {
		t.Errorf("Expected Canceled error, got: %v", gatherErr)
	}
}

// TestCollectorFunc tests the CollectorFunc adapter
func TestCollectorFunc(t *testing.T) {
	var descCalled, collectCalled bool
	var receivedContext context.Context

	cf := NewCollectorFunc(
		func(ch chan<- *prometheus.Desc) {
			descCalled = true
			ch <- prometheus.NewDesc("test_metric", "help", nil, nil)
		},
		func(ctx context.Context, ch chan<- prometheus.Metric) {
			collectCalled = true
			receivedContext = ctx
			metric, _ := prometheus.NewConstMetric(
				prometheus.NewDesc("test_metric", "help", nil, nil),
				prometheus.GaugeValue,
				42.0,
			)
			ch <- metric
		},
	)

	// Test Describe
	descCh := make(chan *prometheus.Desc, 1)
	cf.Describe(descCh)
	close(descCh)

	if !descCalled {
		t.Error("Describe function was not called")
	}

	// Test CollectWithContext
	type testKey string
	const key testKey = "test"
	ctx := context.WithValue(context.Background(), key, "value")
	metricCh := make(chan prometheus.Metric, 1)
	cf.CollectWithContext(ctx, metricCh)
	close(metricCh)

	if !collectCalled {
		t.Error("Collect function was not called")
	}

	if receivedContext.Value(key) != "value" {
		t.Error("Context was not properly passed to collect function")
	}
}

// TestCollectorAdapter tests the adapter for standard collectors
func TestCollectorAdapter(t *testing.T) {
	// Create a standard collector
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "adapted_counter",
		Help: "An adapted counter",
	})

	// Adapt it to be context-aware
	adapted := NewCollectorAdapter(counter)

	// Test that it still works with context
	ctx := context.Background()
	ch := make(chan prometheus.Metric, 1)

	// This should not panic
	adapted.CollectWithContext(ctx, ch)

	// Verify it implements both interfaces
	var _ prometheus.Collector = adapted
	var _ = adapted
}

// TestMultiGathererWithContext tests the multi-gatherer with context support
func TestMultiGathererWithContext(t *testing.T) {
	mg := NewMultiGathererWithContext()

	// Register a standard registry
	reg1 := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "counter1",
		Help: "First counter",
	})
	reg1.MustRegister(counter)

	err := mg.Register("namespace1", reg1)
	if err != nil {
		t.Fatalf("Failed to register first gatherer: %v", err)
	}

	// Register a context-aware registry
	reg2 := NewContextRegistry()
	contextCollector := &testContextCollector{
		name: "context_metric",
		help: "Context-aware metric",
	}
	reg2.MustRegister(contextCollector)

	err = mg.Register("namespace2", reg2)
	if err != nil {
		t.Fatalf("Failed to register second gatherer: %v", err)
	}

	// Gather with context
	ctx := context.Background()
	mfs, err := mg.GatherWithContext(ctx)
	if err != nil {
		t.Fatalf("Failed to gather with context: %v", err)
	}

	// Should have metrics from both registries
	if len(mfs) < 2 {
		t.Errorf("Expected at least 2 metric families, got %d", len(mfs))
	}

	// Debug: print what we got
	for _, mf := range mfs {
		if mf.Name != nil {
			t.Logf("Found metric: %s", *mf.Name)
		}
	}

	// Check that namespaces were applied
	foundNamespace1 := false
	foundNamespace2 := false
	for _, mf := range mfs {
		if mf.Name != nil {
			name := *mf.Name
			// Check if the name starts with the namespace prefix
			if name == "namespace1_counter1" {
				foundNamespace1 = true
			}
			// The context metric might have a different name pattern
			if name == "namespace2_context_metric" || name == "namespace2_test_context_metric" {
				foundNamespace2 = true
			}
		}
	}

	if !foundNamespace1 {
		t.Error("Namespace1 prefix not found")
	}
	if !foundNamespace2 {
		t.Error("Namespace2 prefix not found")
	}
}

// testContextCollector is a test collector that implements CollectorWithContext
type testContextCollector struct {
	name            string
	help            string
	contextReceived bool
	mu              sync.Mutex
}

func (c *testContextCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(c.name, c.help, nil, nil)
}

func (c *testContextCollector) Collect(ch chan<- prometheus.Metric) {
	c.CollectWithContext(context.Background(), ch)
}

func (c *testContextCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	c.mu.Lock()
	c.contextReceived = ctx != nil
	c.mu.Unlock()

	metric, _ := prometheus.NewConstMetric(
		prometheus.NewDesc(c.name, c.help, nil, nil),
		prometheus.GaugeValue,
		1.0,
	)
	ch <- metric
}

// testSlowCollector is a test collector that takes time to collect
type testSlowCollector struct {
	delay time.Duration
	name  string
}

func (c *testSlowCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(c.name, "Slow metric", nil, nil)
}

func (c *testSlowCollector) Collect(ch chan<- prometheus.Metric) {
	time.Sleep(c.delay)
	metric, _ := prometheus.NewConstMetric(
		prometheus.NewDesc(c.name, "Slow metric", nil, nil),
		prometheus.GaugeValue,
		1.0,
	)
	ch <- metric
}

// testCancellableCollector is a test collector that respects cancellation
type testCancellableCollector struct {
	name      string
	cancelled bool
	mu        sync.Mutex
}

func (c *testCancellableCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(c.name, "Cancellable metric", nil, nil)
}

func (c *testCancellableCollector) Collect(ch chan<- prometheus.Metric) {
	c.CollectWithContext(context.Background(), ch)
}

func (c *testCancellableCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	// Simulate some work that can be cancelled
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.cancelled = true
			c.mu.Unlock()
			return
		case <-ticker.C:
			// Continue working
		}
	}

	// If not cancelled, send a metric
	metric, _ := prometheus.NewConstMetric(
		prometheus.NewDesc(c.name, "Cancellable metric", nil, nil),
		prometheus.GaugeValue,
		1.0,
	)
	ch <- metric
}
