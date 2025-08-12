// Copyright (C) 2019-2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"context"
	
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

// GathererWithContext is a context-aware variant of prometheus.Gatherer
// that accepts a context.Context for timeout and cancellation propagation.
type GathererWithContext interface {
	// GatherWithContext works like Gather but accepts a context for
	// timeout/cancellation and request-scoped values.
	GatherWithContext(context.Context) ([]*dto.MetricFamily, error)
}

// CollectorWithContext is a context-aware variant of prometheus.Collector
// that accepts a context.Context in its Collect method.
type CollectorWithContext interface {
	// DescribeByCollect returns true if Describe should simply invoke
	// CollectWithContext and send the descriptors of the gathered metrics.
	//
	// This is useful for custom Collectors that dynamically generate metrics.
	DescribeByCollect() bool
	
	// Describe sends the super-set of all possible descriptors of metrics
	// collected by this Collector to the provided channel and returns once
	// the last descriptor has been sent.
	Describe(chan<- *prometheus.Desc)
	
	// CollectWithContext works like Collect but accepts a context for
	// timeout/cancellation and request-scoped values.
	CollectWithContext(context.Context, chan<- prometheus.Metric)
}

// RegistryWithContext extends the standard Registry to support context-aware
// collectors and gatherers.
type RegistryWithContext struct {
	*prometheus.Registry
	contextCollectors []CollectorWithContext
}

// NewRegistryWithContext creates a new registry that supports both standard
// and context-aware collectors.
func NewRegistryWithContext() *RegistryWithContext {
	return &RegistryWithContext{
		Registry:          prometheus.NewRegistry(),
		contextCollectors: make([]CollectorWithContext, 0),
	}
}

// RegisterWithContext registers a context-aware collector.
func (r *RegistryWithContext) RegisterWithContext(c CollectorWithContext) error {
	r.contextCollectors = append(r.contextCollectors, c)
	// Also register a wrapper for backward compatibility
	return r.Register(&contextCollectorWrapper{collector: c})
}

// GatherWithContext implements GathererWithContext interface.
func (r *RegistryWithContext) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
	// First gather from standard collectors
	families, err := r.Gather()
	if err != nil {
		return nil, err
	}
	
	// Then gather from context-aware collectors
	for _, collector := range r.contextCollectors {
		ch := make(chan prometheus.Metric, 100)
		go func() {
			collector.CollectWithContext(ctx, ch)
			close(ch)
		}()
		
		// Collect metrics from channel
		for metric := range ch {
			// Convert metric to dto.MetricFamily
			// This is simplified - real implementation would need proper conversion
			desc := metric.Desc()
			family := &dto.MetricFamily{
				Name: proto.String(desc.String()),
				Help: proto.String("Context-aware metric"),
				Type: dto.MetricType_GAUGE.Enum(),
			}
			families = append(families, family)
		}
	}
	
	return families, nil
}

// contextCollectorWrapper wraps a CollectorWithContext to implement prometheus.Collector
type contextCollectorWrapper struct {
	collector CollectorWithContext
}

func (w *contextCollectorWrapper) Describe(ch chan<- *prometheus.Desc) {
	w.collector.Describe(ch)
}

func (w *contextCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
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