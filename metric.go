// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Counter is a metric that can only increase
type Counter interface {
	prometheus.Collector
	// Inc increments the counter by 1
	Inc()
	// Add increments the counter by the given value
	Add(float64)
	// Get returns the current value
	Get() float64
}

// Gauge is a metric that can increase or decrease
type Gauge interface {
	prometheus.Collector
	// Set sets the gauge to the given value
	Set(float64)
	// Inc increments the gauge by 1
	Inc()
	// Dec decrements the gauge by 1
	Dec()
	// Add adds the given value to the gauge
	Add(float64)
	// Sub subtracts the given value from the gauge
	Sub(float64)
	// Get returns the current value
	Get() float64
}

// Histogram samples observations and counts them in configurable buckets
type Histogram interface {
	prometheus.Collector
	// Observe adds a single observation to the histogram
	Observe(float64)
}

// Summary captures individual observations and provides quantiles
type Summary interface {
	prometheus.Collector
	// Observe adds a single observation to the summary
	Observe(float64)
}

// Timer measures durations
type Timer interface {
	// Start starts the timer and returns a function to stop it
	Start() func()
	// ObserveTime observes the given duration
	ObserveTime(time.Duration)
}

// Labels represents a set of label key-value pairs
type Labels map[string]string

// Registerer is an alias for prometheus.Registerer
type Registerer = prometheus.Registerer

// Gatherer is an alias for prometheus.Gatherer
type Gatherer = prometheus.Gatherer

// MetricFamily alias for dto.MetricFamily
type MetricFamily = dto.MetricFamily

// Registry is an alias for prometheus.Registry to keep it internal
// We use prometheus.Registry directly but alias it to avoid external dependencies
type Registry = *prometheus.Registry

// Collector is an alias for prometheus.Collector
type Collector = prometheus.Collector

// Metric is an alias for prometheus.Metric
type Metric = prometheus.Metric

// Desc is an alias for prometheus.Desc
type Desc = *prometheus.Desc

// Metrics is the main interface for creating metrics
type Metrics interface {
	// NewCounter creates a new counter
	NewCounter(name, help string) Counter
	// NewCounterVec creates a new counter vector
	NewCounterVec(name, help string, labelNames []string) CounterVec

	// NewGauge creates a new gauge
	NewGauge(name, help string) Gauge
	// NewGaugeVec creates a new gauge vector
	NewGaugeVec(name, help string, labelNames []string) GaugeVec

	// NewHistogram creates a new histogram
	NewHistogram(name, help string, buckets []float64) Histogram
	// NewHistogramVec creates a new histogram vector
	NewHistogramVec(name, help string, labelNames []string, buckets []float64) HistogramVec

	// NewSummary creates a new summary
	NewSummary(name, help string, objectives map[float64]float64) Summary
	// NewSummaryVec creates a new summary vector
	NewSummaryVec(name, help string, labelNames []string, objectives map[float64]float64) SummaryVec

	// Registry returns the underlying registry
	Registry() Registry

	// PrometheusRegistry returns the prometheus registerer for compatibility
	PrometheusRegistry() interface{}
}

// CounterVec is a vector of counters
type CounterVec interface {
	prometheus.Collector
	// With returns a counter with the given label values
	With(Labels) Counter
	// WithLabelValues returns a counter with the given label values
	WithLabelValues(labelValues ...string) Counter
}

// GaugeVec is a vector of gauges
type GaugeVec interface {
	prometheus.Collector
	// With returns a gauge with the given label values
	With(Labels) Gauge
	// WithLabelValues returns a gauge with the given label values
	WithLabelValues(labelValues ...string) Gauge
}

// HistogramVec is a vector of histograms
type HistogramVec interface {
	prometheus.Collector
	// With returns a histogram with the given label values
	With(Labels) Histogram
	// WithLabelValues returns a histogram with the given label values
	WithLabelValues(labelValues ...string) Histogram
}

// SummaryVec is a vector of summaries
type SummaryVec interface {
	// With returns a summary with the given label values
	With(Labels) Summary
	// WithLabelValues returns a summary with the given label values
	WithLabelValues(labelValues ...string) Summary
}

// Factory creates new metrics instances
type Factory interface {
	// New creates a new metrics instance with the given namespace
	New(namespace string) Metrics
	// NewWithRegistry creates a new metrics instance with a custom registry
	NewWithRegistry(namespace string, registry Registry) Metrics
}

// MetricsHTTPHandler handles HTTP requests for metrics
type MetricsHTTPHandler interface {
	// ServeHTTP handles an HTTP request
	ServeHTTP(w ResponseWriter, r *Request)
}

// ResponseWriter is an interface for writing HTTP responses
type ResponseWriter interface {
	// Write writes data to the response
	Write([]byte) (int, error)
	// WriteHeader writes the status code
	WriteHeader(int)
	// Header returns the response headers
	Header() map[string][]string
}

// Request represents an HTTP request
type Request interface {
	// Context returns the request context
	Context() context.Context
	// Method returns the HTTP method
	Method() string
	// URL returns the request URL
	URL() string
}

// Global factory instance
var defaultFactory Factory = NewPrometheusFactory()

// SetFactory sets the global metrics factory
func SetFactory(factory Factory) {
	defaultFactory = factory
}

// New creates a new metrics instance with the given namespace
func New(namespace string) Metrics {
	return defaultFactory.New(namespace)
}

// NewWithRegistry creates a new metrics instance with a custom registry
func NewWithRegistry(namespace string, registry Registry) Metrics {
	return defaultFactory.NewWithRegistry(namespace, registry)
}

// Export prometheus types
type (
	CounterOpts   = prometheus.CounterOpts
	GaugeOpts     = prometheus.GaugeOpts  
	HistogramOpts = prometheus.HistogramOpts
	SummaryOpts   = prometheus.SummaryOpts
	Gatherers     = prometheus.Gatherers
)

// Constructor functions that return wrapped types
func NewCounter(opts CounterOpts) Counter {
	return WrapPrometheusCounter(prometheus.NewCounter(opts))
}

func NewCounterVec(opts CounterOpts, labelNames []string) CounterVec {
	return WrapPrometheusCounterVec(prometheus.NewCounterVec(opts, labelNames))
}

func NewGauge(opts GaugeOpts) Gauge {
	return WrapPrometheusGauge(prometheus.NewGauge(opts))
}

func NewGaugeVec(opts GaugeOpts, labelNames []string) GaugeVec {
	return WrapPrometheusGaugeVec(prometheus.NewGaugeVec(opts, labelNames))
}

func NewHistogramVec(opts HistogramOpts, labelNames []string) HistogramVec {
	return WrapPrometheusHistogramVec(prometheus.NewHistogramVec(opts, labelNames))
}

// Keep these as direct aliases since they don't need wrapping
var (
	NewHistogram       = prometheus.NewHistogram
	NewSummary         = prometheus.NewSummary
	NewSummaryVec      = prometheus.NewSummaryVec
	NewRegistry        = prometheus.NewRegistry
	NewDesc            = prometheus.NewDesc
	MustNewConstMetric = prometheus.MustNewConstMetric
	Register           = prometheus.Register
	MustRegister       = prometheus.MustRegister
	Unregister         = prometheus.Unregister
)
