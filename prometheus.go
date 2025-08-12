// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// prometheusMetrics implements Metrics using prometheus
type prometheusMetrics struct {
	namespace string
	registry  *prometheus.Registry
}

func (p *prometheusMetrics) NewCounter(name, help string) Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	})
	p.registry.MustRegister(counter)
	return counter
}

func (p *prometheusMetrics) NewCounterVec(name, help string, labelNames []string) CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	p.registry.MustRegister(vec)
	return vec
}

func (p *prometheusMetrics) NewGauge(name, help string) Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	})
	p.registry.MustRegister(gauge)
	return gauge
}

func (p *prometheusMetrics) NewGaugeVec(name, help string, labelNames []string) GaugeVec {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	p.registry.MustRegister(vec)
	return vec
}

func (p *prometheusMetrics) NewHistogram(name, help string, buckets []float64) Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})
	p.registry.MustRegister(histogram)
	return histogram
}

func (p *prometheusMetrics) NewHistogramVec(name, help string, labelNames []string, buckets []float64) HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labelNames)
	p.registry.MustRegister(vec)
	return vec
}

func (p *prometheusMetrics) NewSummary(name, help string, objectives map[float64]float64) Summary {
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  p.namespace,
		Name:       name,
		Help:       help,
		Objectives: objectives,
	})
	p.registry.MustRegister(summary)
	return summary
}

func (p *prometheusMetrics) NewSummaryVec(name, help string, labelNames []string, objectives map[float64]float64) SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  p.namespace,
		Name:       name,
		Help:       help,
		Objectives: objectives,
	}, labelNames)
	p.registry.MustRegister(vec)
	return vec
}

func (p *prometheusMetrics) Registry() Registry {
	return p.registry
}

func (p *prometheusMetrics) PrometheusRegistry() prometheus.Registerer {
	return p.registry
}

// prometheusFactory creates prometheus-backed metrics
type prometheusFactory struct {
	defaultRegistry *prometheus.Registry
}

// NewPrometheusFactory creates a factory that produces prometheus-backed metrics
func NewPrometheusFactory() Factory {
	return &prometheusFactory{
		defaultRegistry: prometheus.NewRegistry(),
	}
}

// NewPrometheusFactoryWithRegistry creates a factory with a custom prometheus registry
func NewPrometheusFactoryWithRegistry(registry *prometheus.Registry) Factory {
	return &prometheusFactory{
		defaultRegistry: registry,
	}
}

func (f *prometheusFactory) New(namespace string) Metrics {
	return &prometheusMetrics{
		namespace: namespace,
		registry:  f.defaultRegistry,
	}
}

func (f *prometheusFactory) NewWithRegistry(namespace string, registry Registry) Metrics {
	// Registry is already *prometheus.Registry, use it directly
	return &prometheusMetrics{
		namespace: namespace,
		registry:  registry,
	}
}

// NewPrometheusMetrics creates a new prometheus-backed metrics instance
func NewPrometheusMetrics(namespace string, registry *prometheus.Registry) Metrics {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	return &prometheusMetrics{
		namespace: namespace,
		registry:  registry,
	}
}