// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	// PrometheusRegistry returns the underlying Prometheus registry, if any
	PrometheusRegistry() interface{}
}

// Factory creates new metrics instances
type Factory interface {
	// New creates a new metrics instance with the given namespace
	New(namespace string) Metrics
	// NewWithRegistry creates a new metrics instance with a custom registry
	NewWithRegistry(namespace string, registry Registry) Metrics
}

// CounterVec is a labeled counter collection
type CounterVec interface {
	With(Labels) Counter
	WithLabelValues(...string) Counter
}

// GaugeVec is a labeled gauge collection
type GaugeVec interface {
	With(Labels) Gauge
	WithLabelValues(...string) Gauge
}

// HistogramVec is a labeled histogram collection
type HistogramVec interface {
	With(Labels) Histogram
	WithLabelValues(...string) Histogram
}

// SummaryVec is a labeled summary collection
type SummaryVec interface {
	With(Labels) Summary
	WithLabelValues(...string) Summary
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
var defaultFactory Factory = NewHighPerfMetricsFactory()

// SetFactory sets the global metrics factory
func SetFactory(factory Factory) {
	defaultFactory = factory
}

// New creates a new metrics instance with the given namespace
func New(namespace string) Metrics {
	return defaultFactory.New(namespace)
}

// NewWithRegistry creates a new metrics instance with the provided registry.
// If registry is nil, it falls back to the default factory.
func NewWithRegistry(namespace string, registry Registry) Metrics {
	if registry == nil {
		return New(namespace)
	}
	return NewPrometheusFactoryWithRegistry(registry).New(namespace)
}

// NewCounter creates a new high-performance counter
func NewCounter(opts CounterOpts) Counter {
	return NewCounterWithOpts(opts)
}

// NewGauge creates a new high-performance gauge
func NewGauge(opts GaugeOpts) Gauge {
	return NewGaugeWithOpts(opts)
}

// NewHistogram creates a new high-performance histogram
func NewHistogram(opts HistogramOpts) Histogram {
	return WrapPrometheusHistogram(prometheus.NewHistogram(opts))
}

// NewSummary creates a new high-performance summary
func NewSummary(opts SummaryOpts) Summary {
	return WrapPrometheusSummary(prometheus.NewSummary(opts))
}
