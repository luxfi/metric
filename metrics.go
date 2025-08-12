// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"context"
	"net/http"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

// Counter is a metric that can only increase
type Counter interface {
	// Inc increments the counter by 1
	Inc()
	// Add increments the counter by the given value
	Add(float64)
	// Get returns the current value
	Get() float64
}

// Gauge is a metric that can increase or decrease
type Gauge interface {
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
	// Observe adds a single observation to the histogram
	Observe(float64)
}

// Summary captures individual observations and provides quantiles
type Summary interface {
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
}

// CounterVec is a vector of counters
type CounterVec interface {
	// With returns a counter with the given label values
	With(Labels) Counter
	// WithLabelValues returns a counter with the given label values
	WithLabelValues(labelValues ...string) Counter
}

// GaugeVec is a vector of gauges
type GaugeVec interface {
	// With returns a gauge with the given label values
	With(Labels) Gauge
	// WithLabelValues returns a gauge with the given label values
	WithLabelValues(labelValues ...string) Gauge
}

// HistogramVec is a vector of histograms
type HistogramVec interface {
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

// NewPrometheusRegistry creates a new prometheus registry
func NewPrometheusRegistry() Registry {
	return prometheus.NewRegistry()
}

// PrometheusRegistry is an alias for prometheus.Registry
type PrometheusRegistry = prometheus.Registry

// HTTPHandler creates an HTTP handler for metrics
func HTTPHandler(gatherer prometheus.Gatherer, opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(gatherer, opts)
}

// HTTPHandlerOpts are options for the HTTP handler
type HTTPHandlerOpts = promhttp.HandlerOpts

// ProcessCollectorOpts are options for the process collector
type ProcessCollectorOpts = collectors.ProcessCollectorOpts

// NewProcessCollector creates a new process collector
func NewProcessCollector(opts ProcessCollectorOpts) prometheus.Collector {
	return collectors.NewProcessCollector(opts)
}

// NewGoCollector creates a new Go collector
func NewGoCollector() prometheus.Collector {
	return collectors.NewGoCollector()
}

