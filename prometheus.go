// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

// prometheusCounter wraps prometheus.Counter
type prometheusCounter struct {
	counter prometheus.Counter
}

func (p *prometheusCounter) Inc()          { p.counter.Inc() }
func (p *prometheusCounter) Add(v float64) { p.counter.Add(v) }
func (p *prometheusCounter) Get() float64  { return 0 } // Prometheus doesn't expose current value

// Implement prometheus.Collector interface
func (p *prometheusCounter) Describe(ch chan<- *prometheus.Desc) { p.counter.Describe(ch) }
func (p *prometheusCounter) Collect(ch chan<- prometheus.Metric)  { p.counter.Collect(ch) }

// prometheusGauge wraps prometheus.Gauge
type prometheusGauge struct {
	gauge prometheus.Gauge
}

func (p *prometheusGauge) Set(v float64) { p.gauge.Set(v) }
func (p *prometheusGauge) Inc()          { p.gauge.Inc() }
func (p *prometheusGauge) Dec()          { p.gauge.Dec() }
func (p *prometheusGauge) Add(v float64) { p.gauge.Add(v) }
func (p *prometheusGauge) Sub(v float64) { p.gauge.Sub(v) }
func (p *prometheusGauge) Get() float64  { return 0 } // Prometheus doesn't expose current value

// Implement prometheus.Collector interface
func (p *prometheusGauge) Describe(ch chan<- *prometheus.Desc) { p.gauge.Describe(ch) }
func (p *prometheusGauge) Collect(ch chan<- prometheus.Metric)  { p.gauge.Collect(ch) }

// prometheusHistogram wraps prometheus.Histogram
type prometheusHistogram struct {
	histogram prometheus.Histogram
}

func (p *prometheusHistogram) Observe(v float64) { p.histogram.Observe(v) }

// Implement prometheus.Collector interface
func (p *prometheusHistogram) Describe(ch chan<- *prometheus.Desc) { p.histogram.Describe(ch) }
func (p *prometheusHistogram) Collect(ch chan<- prometheus.Metric)  { p.histogram.Collect(ch) }

// prometheusSummary wraps prometheus.Summary
type prometheusSummary struct {
	summary prometheus.Summary
}

func (p *prometheusSummary) Observe(v float64) { p.summary.Observe(v) }

// Implement prometheus.Collector interface
func (p *prometheusSummary) Describe(ch chan<- *prometheus.Desc) { p.summary.Describe(ch) }
func (p *prometheusSummary) Collect(ch chan<- prometheus.Metric)  { p.summary.Collect(ch) }

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

// Implement prometheus.Collector interface
func (p *prometheusCounterVec) Describe(ch chan<- *prometheus.Desc) { p.vec.Describe(ch) }
func (p *prometheusCounterVec) Collect(ch chan<- prometheus.Metric)  { p.vec.Collect(ch) }

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

// Implement prometheus.Collector interface
func (p *prometheusGaugeVec) Describe(ch chan<- *prometheus.Desc) { p.vec.Describe(ch) }
func (p *prometheusGaugeVec) Collect(ch chan<- prometheus.Metric)  { p.vec.Collect(ch) }

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
	return p.registry
}

func (p *prometheusMetrics) PrometheusRegistry() interface{} {
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
