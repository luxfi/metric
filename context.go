// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

// CollectorWithContext is a Collector that can consume a Context for
// timeout/cancellation and request-scoped values.
type CollectorWithContext interface {
	prometheus.Collector // Embeds the base interface for compatibility

	// CollectWithContext works like Collect but accepts a context for
	// timeout/cancellation and request-scoped values.
	CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric)
}

// GathererWithContext is a Gatherer that accepts a context.Context for
// timeout/cancellation propagation and request-scoped values.
type GathererWithContext interface {
	prometheus.Gatherer // Embeds the base interface for compatibility

	// GatherWithContext works like Gather but accepts a context for
	// timeout/cancellation and request-scoped values.
	GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error)
}

// ContextRegistry is a registry that supports both standard and context-aware collectors.
// It implements both prometheus.Gatherer and GathererWithContext interfaces.
type ContextRegistry struct {
	mu         sync.RWMutex
	collectors map[uint64]collectorEntry // Map of collector ID to entry
	nextID     uint64
	pedantic   bool // If true, perform extra validation
}

// collectorEntry stores a collector along with metadata
type collectorEntry struct {
	id        uint64
	collector prometheus.Collector
	isContext bool // True if this collector implements CollectorWithContext
}

// NewContextRegistry creates a new registry that supports both standard
// and context-aware collectors.
func NewContextRegistry() *ContextRegistry {
	return &ContextRegistry{
		collectors: make(map[uint64]collectorEntry),
		pedantic:   false,
	}
}

// NewPedanticContextRegistry creates a new registry with extra validation enabled.
func NewPedanticContextRegistry() *ContextRegistry {
	r := NewContextRegistry()
	r.pedantic = true
	return r
}

// Register registers a collector (either standard or context-aware).
func (r *ContextRegistry) Register(c prometheus.Collector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if it's a context-aware collector
	_, isContext := c.(CollectorWithContext)

	// Get descriptors to validate the collector
	descChan := make(chan *prometheus.Desc, 16)
	go func() {
		c.Describe(descChan)
		close(descChan)
	}()

	// Collect all descriptors
	var descs []*prometheus.Desc
	for desc := range descChan {
		descs = append(descs, desc)
	}

	// If pedantic mode, check for duplicates
	if r.pedantic {
		for _, existing := range r.collectors {
			existingDescChan := make(chan *prometheus.Desc, 16)
			go func() {
				existing.collector.Describe(existingDescChan)
				close(existingDescChan)
			}()

			for existingDesc := range existingDescChan {
				for _, newDesc := range descs {
					if existingDesc.String() == newDesc.String() {
						return fmt.Errorf("descriptor %s already registered", newDesc)
					}
				}
			}
		}
	}

	// Register the collector
	entry := collectorEntry{
		id:        r.nextID,
		collector: c,
		isContext: isContext,
	}
	r.collectors[r.nextID] = entry
	r.nextID++

	return nil
}

// MustRegister registers collectors and panics on error.
func (r *ContextRegistry) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := r.Register(c); err != nil {
			panic(err)
		}
	}
}

// Unregister removes a collector from the registry.
func (r *ContextRegistry) Unregister(c prometheus.Collector) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, entry := range r.collectors {
		if entry.collector == c {
			delete(r.collectors, id)
			return true
		}
	}
	return false
}

// Gather implements prometheus.Gatherer for backward compatibility.
func (r *ContextRegistry) Gather() ([]*dto.MetricFamily, error) {
	return r.GatherWithContext(context.Background())
}

// GatherWithContext implements GathererWithContext interface.
func (r *ContextRegistry) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
	r.mu.RLock()
	collectors := make([]collectorEntry, 0, len(r.collectors))
	for _, entry := range r.collectors {
		collectors = append(collectors, entry)
	}
	r.mu.RUnlock()

	// Check if context is already canceled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Channel for receiving metrics from collectors
	metricCh := make(chan prometheus.Metric, 1024)
	var wg sync.WaitGroup

	// Error channel for collector errors
	errCh := make(chan error, len(collectors))

	// Start collectors in parallel
	for _, entry := range collectors {
		// Check context before starting each collector
		if ctx.Err() != nil {
			close(metricCh)
			return nil, ctx.Err()
		}

		wg.Add(1)
		go func(e collectorEntry) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errCh <- fmt.Errorf("collector panicked: %v", r)
				}
			}()

			// Create a collector-specific channel
			collectorCh := make(chan prometheus.Metric, 256)
			done := make(chan struct{})

			// Forward metrics from collector channel to main channel
			go func() {
				for metric := range collectorCh {
					select {
					case metricCh <- metric:
					case <-ctx.Done():
						return
					}
				}
				close(done)
			}()

			// Call the appropriate collect method
			if e.isContext {
				if cwc, ok := e.collector.(CollectorWithContext); ok {
					cwc.CollectWithContext(ctx, collectorCh)
				}
			} else {
				e.collector.Collect(collectorCh)
			}

			close(collectorCh)
			<-done
		}(entry)
	}

	// Wait for all collectors to finish and close the metric channel
	go func() {
		wg.Wait()
		close(metricCh)
		close(errCh)
	}()

	// Collect metrics into families
	metricFamilies := make(map[string]*dto.MetricFamily)

	for metric := range metricCh {
		// Check context periodically
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Convert metric to DTO
		dtoMetric := &dto.Metric{}
		if err := metric.Write(dtoMetric); err != nil {
			return nil, fmt.Errorf("error writing metric: %w", err)
		}

		// Get metric descriptor
		desc := metric.Desc()

		// Extract the fully qualified name from the descriptor
		// We need to parse the descriptor string to get the actual metric name
		// The descriptor string format is: Desc{fqName: "name", help: "...", ...}
		descString := desc.String()
		var fqName string

		// Parse the fqName from the descriptor string
		if idx := strings.Index(descString, `fqName: "`); idx >= 0 {
			start := idx + len(`fqName: "`)
			if end := strings.Index(descString[start:], `"`); end >= 0 {
				fqName = descString[start : start+end]
			}
		}

		// If we couldn't parse it, use the whole string as fallback
		if fqName == "" {
			fqName = descString
		}

		// Get or create metric family
		mf, exists := metricFamilies[fqName]
		if !exists {
			mf = &dto.MetricFamily{
				Name: proto.String(fqName),
				Help: proto.String(""),              // Would be extracted from descriptor
				Type: dto.MetricType_UNTYPED.Enum(), // Would be determined from metric type
			}
			metricFamilies[fqName] = mf
		}

		// Add metric to family
		mf.Metric = append(mf.Metric, dtoMetric)
	}

	// Check for collector errors
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	// Convert map to sorted slice
	result := make([]*dto.MetricFamily, 0, len(metricFamilies))
	for _, mf := range metricFamilies {
		result = append(result, mf)
	}

	// Sort by name for consistent output
	sort.Slice(result, func(i, j int) bool {
		return *result[i].Name < *result[j].Name
	})

	return result, nil
}

// CollectorFunc is an adapter to allow the use of ordinary functions as
// context-aware collectors.
type CollectorFunc struct {
	descFunc    func(chan<- *prometheus.Desc)
	collectFunc func(context.Context, chan<- prometheus.Metric)
}

// NewCollectorFunc creates a new CollectorFunc.
func NewCollectorFunc(
	descFunc func(chan<- *prometheus.Desc),
	collectFunc func(context.Context, chan<- prometheus.Metric),
) *CollectorFunc {
	return &CollectorFunc{
		descFunc:    descFunc,
		collectFunc: collectFunc,
	}
}

// Describe implements prometheus.Collector.
func (f *CollectorFunc) Describe(ch chan<- *prometheus.Desc) {
	if f.descFunc != nil {
		f.descFunc(ch)
	}
}

// Collect implements prometheus.Collector.
func (f *CollectorFunc) Collect(ch chan<- prometheus.Metric) {
	if f.collectFunc != nil {
		f.collectFunc(context.Background(), ch)
	}
}

// CollectWithContext implements CollectorWithContext.
func (f *CollectorFunc) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	if f.collectFunc != nil {
		f.collectFunc(ctx, ch)
	}
}

// CollectorAdapter wraps a standard prometheus.Collector to implement CollectorWithContext.
type CollectorAdapter struct {
	prometheus.Collector
}

// NewCollectorAdapter creates a new adapter for a standard collector.
func NewCollectorAdapter(c prometheus.Collector) CollectorWithContext {
	return &CollectorAdapter{Collector: c}
}

// CollectWithContext implements CollectorWithContext by ignoring the context.
func (a *CollectorAdapter) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	// Simply delegate to the standard Collect method
	a.Collect(ch)
}

// ContextCollectorWrapper wraps a CollectorWithContext to implement prometheus.Collector.
type ContextCollectorWrapper struct {
	collector CollectorWithContext
}

// NewContextCollectorWrapper creates a wrapper for a context-aware collector.
func NewContextCollectorWrapper(c CollectorWithContext) prometheus.Collector {
	return &ContextCollectorWrapper{collector: c}
}

// Describe implements prometheus.Collector.
func (w *ContextCollectorWrapper) Describe(ch chan<- *prometheus.Desc) {
	w.collector.Describe(ch)
}

// Collect implements prometheus.Collector.
func (w *ContextCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	// Use background context when called without context
	w.collector.CollectWithContext(context.Background(), ch)
}

// GathererFunc is an adapter to allow the use of ordinary functions as Gatherers.
type GathererFunc func() ([]*dto.MetricFamily, error)

// Gather implements prometheus.Gatherer.
func (f GathererFunc) Gather() ([]*dto.MetricFamily, error) {
	return f()
}

// GathererWithContextFunc is an adapter to allow the use of ordinary functions
// as context-aware Gatherers.
type GathererWithContextFunc func(context.Context) ([]*dto.MetricFamily, error)

// GatherWithContext implements GathererWithContext.
func (f GathererWithContextFunc) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
	return f(ctx)
}

// Gather implements prometheus.Gatherer for backward compatibility.
func (f GathererWithContextFunc) Gather() ([]*dto.MetricFamily, error) {
	return f(context.Background())
}

// MultiGathererWithContext extends MultiGatherer with context support.
type MultiGathererWithContext interface {
	GathererWithContext

	// Register adds the outputs of gatherer to the results of future calls to
	// Gather with the provided namespace added to the metrics.
	Register(namespace string, gatherer prometheus.Gatherer) error

	// Deregister removes the outputs of a gatherer with namespace from the results
	// of future calls to Gather.
	Deregister(namespace string) bool
}

// multiGathererWithContext implements MultiGathererWithContext.
type multiGathererWithContext struct {
	mu        sync.RWMutex
	gatherers map[string]prometheus.Gatherer
}

// NewMultiGathererWithContext creates a new MultiGathererWithContext.
func NewMultiGathererWithContext() MultiGathererWithContext {
	return &multiGathererWithContext{
		gatherers: make(map[string]prometheus.Gatherer),
	}
}

// Register adds a gatherer with a namespace.
func (g *multiGathererWithContext) Register(namespace string, gatherer prometheus.Gatherer) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.gatherers[namespace]; exists {
		return fmt.Errorf("gatherer already registered for namespace: %s", namespace)
	}
	g.gatherers[namespace] = gatherer
	return nil
}

// Deregister removes a gatherer by namespace.
func (g *multiGathererWithContext) Deregister(namespace string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	_, exists := g.gatherers[namespace]
	delete(g.gatherers, namespace)
	return exists
}

// Gather implements prometheus.Gatherer.
func (g *multiGathererWithContext) Gather() ([]*dto.MetricFamily, error) {
	return g.GatherWithContext(context.Background())
}

// GatherWithContext implements GathererWithContext.
func (g *multiGathererWithContext) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
	g.mu.RLock()
	gatherers := make(map[string]prometheus.Gatherer, len(g.gatherers))
	for k, v := range g.gatherers {
		gatherers[k] = v
	}
	g.mu.RUnlock()

	var result []*dto.MetricFamily
	for namespace, gatherer := range gatherers {
		// Check context before gathering from each gatherer
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var metrics []*dto.MetricFamily
		var err error

		// Use context-aware gathering if available
		if gwc, ok := gatherer.(GathererWithContext); ok {
			metrics, err = gwc.GatherWithContext(ctx)
		} else {
			metrics, err = gatherer.Gather()
		}

		if err != nil {
			return nil, fmt.Errorf("error gathering from namespace %s: %w", namespace, err)
		}

		// Add namespace prefix to each metric
		for _, mf := range metrics {
			prefixedName := namespace + "_" + *mf.Name
			mf.Name = &prefixedName
		}

		result = append(result, metrics...)
	}

	return result, nil
}
