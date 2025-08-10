// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// prometheusCounter wraps prometheus.Counter
type prometheusCounter struct {
	counter prometheus.Counter
}

func (p *prometheusCounter) Inc()         { p.counter.Inc() }
func (p *prometheusCounter) Add(v float64) { p.counter.Add(v) }
func (p *prometheusCounter) Get() float64 { 
	// Prometheus doesn't expose current value directly
	// This is a limitation of the prometheus client
	return 0 
}

// prometheusGauge wraps prometheus.Gauge
type prometheusGauge struct {
	gauge prometheus.Gauge
}

func (p *prometheusGauge) Set(v float64)  { p.gauge.Set(v) }
func (p *prometheusGauge) Inc()           { p.gauge.Inc() }
func (p *prometheusGauge) Dec()           { p.gauge.Dec() }
func (p *prometheusGauge) Add(v float64)  { p.gauge.Add(v) }
func (p *prometheusGauge) Sub(v float64)  { p.gauge.Sub(v) }
func (p *prometheusGauge) Get() float64   { 
	// Prometheus doesn't expose current value directly
	return 0 
}

// prometheusHistogram wraps prometheus.Histogram
type prometheusHistogram struct {
	histogram prometheus.Histogram
}

func (p *prometheusHistogram) Observe(v float64) { p.histogram.Observe(v) }

// prometheusSummary wraps prometheus.Summary
type prometheusSummary struct {
	summary prometheus.Summary
}

func (p *prometheusSummary) Observe(v float64) { p.summary.Observe(v) }

// prometheusTimer wraps prometheus.Timer
type prometheusTimer struct {
	histogram prometheus.Histogram
}

func (p *prometheusTimer) Start() func() {
	start := time.Now()
	return func() {
		p.histogram.Observe(time.Since(start).Seconds())
	}
}

func (p *prometheusTimer) ObserveTime(d time.Duration) {
	p.histogram.Observe(d.Seconds())
}

// prometheusCounterVec wraps prometheus.CounterVec
type prometheusCounterVec struct {
	vec *prometheus.CounterVec
}

func (p *prometheusCounterVec) With(labels Labels) Counter {
	return &prometheusCounter{counter: p.vec.With(prometheus.Labels(labels))}
}

func (p *prometheusCounterVec) WithLabelValues(labelValues ...string) Counter {
	return &prometheusCounter{counter: p.vec.WithLabelValues(labelValues...)}
}

// prometheusGaugeVec wraps prometheus.GaugeVec
type prometheusGaugeVec struct {
	vec *prometheus.GaugeVec
}

func (p *prometheusGaugeVec) With(labels Labels) Gauge {
	return &prometheusGauge{gauge: p.vec.With(prometheus.Labels(labels))}
}

func (p *prometheusGaugeVec) WithLabelValues(labelValues ...string) Gauge {
	return &prometheusGauge{gauge: p.vec.WithLabelValues(labelValues...)}
}

// prometheusHistogramVec wraps prometheus.HistogramVec
type prometheusHistogramVec struct {
	vec *prometheus.HistogramVec
}

func (p *prometheusHistogramVec) With(labels Labels) Histogram {
	return &prometheusHistogram{histogram: p.vec.With(prometheus.Labels(labels)).(prometheus.Histogram)}
}

func (p *prometheusHistogramVec) WithLabelValues(labelValues ...string) Histogram {
	return &prometheusHistogram{histogram: p.vec.WithLabelValues(labelValues...).(prometheus.Histogram)}
}

// prometheusSummaryVec wraps prometheus.SummaryVec
type prometheusSummaryVec struct {
	vec *prometheus.SummaryVec
}

func (p *prometheusSummaryVec) With(labels Labels) Summary {
	return &prometheusSummary{summary: p.vec.With(prometheus.Labels(labels)).(prometheus.Summary)}
}

func (p *prometheusSummaryVec) WithLabelValues(labelValues ...string) Summary {
	return &prometheusSummary{summary: p.vec.WithLabelValues(labelValues...).(prometheus.Summary)}
}

// prometheusRegistry wraps prometheus.Registry
type prometheusRegistry struct {
	registry *prometheus.Registry
}

func (p *prometheusRegistry) Register(c Collector) error {
	// Note: This requires a type assertion or adapter
	// In practice, you'd need to implement a proper adapter
	return nil
}

func (p *prometheusRegistry) MustRegister(c Collector) {
	// Note: This requires a type assertion or adapter
}

func (p *prometheusRegistry) Unregister(c Collector) bool {
	// Note: This requires a type assertion or adapter
	return true
}

func (p *prometheusRegistry) Gather() ([]*MetricFamily, error) {
	// Note: This would need to convert from prometheus format
	return nil, nil
}

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
	return &prometheusCounter{counter: counter}
}

func (p *prometheusMetrics) NewCounterVec(name, help string, labelNames []string) CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	p.registry.MustRegister(vec)
	return &prometheusCounterVec{vec: vec}
}

func (p *prometheusMetrics) NewGauge(name, help string) Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	})
	p.registry.MustRegister(gauge)
	return &prometheusGauge{gauge: gauge}
}

func (p *prometheusMetrics) NewGaugeVec(name, help string, labelNames []string) GaugeVec {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
	}, labelNames)
	p.registry.MustRegister(vec)
	return &prometheusGaugeVec{vec: vec}
}

func (p *prometheusMetrics) NewHistogram(name, help string, buckets []float64) Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})
	p.registry.MustRegister(histogram)
	return &prometheusHistogram{histogram: histogram}
}

func (p *prometheusMetrics) NewHistogramVec(name, help string, labelNames []string, buckets []float64) HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labelNames)
	p.registry.MustRegister(vec)
	return &prometheusHistogramVec{vec: vec}
}

func (p *prometheusMetrics) NewSummary(name, help string, objectives map[float64]float64) Summary {
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  p.namespace,
		Name:       name,
		Help:       help,
		Objectives: objectives,
	})
	p.registry.MustRegister(summary)
	return &prometheusSummary{summary: summary}
}

func (p *prometheusMetrics) NewSummaryVec(name, help string, labelNames []string, objectives map[float64]float64) SummaryVec {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  p.namespace,
		Name:       name,
		Help:       help,
		Objectives: objectives,
	}, labelNames)
	p.registry.MustRegister(vec)
	return &prometheusSummaryVec{vec: vec}
}

func (p *prometheusMetrics) Registry() Registry {
	return &prometheusRegistry{registry: p.registry}
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
	// Note: This would need to handle type conversion
	promReg, ok := registry.(*prometheusRegistry)
	if ok {
		return &prometheusMetrics{
			namespace: namespace,
			registry:  promReg.registry,
		}
	}
	// Fall back to default registry
	return &prometheusMetrics{
		namespace: namespace,
		registry:  f.defaultRegistry,
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

// PrometheusHandler returns an HTTP handler for prometheus metrics
func PrometheusHandler(registry *prometheus.Registry) http.Handler {
	if registry == nil {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// WrapPrometheusRegistry wraps a prometheus registry in our Registry interface
func WrapPrometheusRegistry(promReg *prometheus.Registry) Registry {
	return &prometheusRegistry{registry: promReg}
}

// UnwrapPrometheusRegistry extracts the prometheus registry from our Registry interface
func UnwrapPrometheusRegistry(reg Registry) (*prometheus.Registry, bool) {
	promReg, ok := reg.(*prometheusRegistry)
	if !ok {
		return nil, false
	}
	return promReg.registry, true
}