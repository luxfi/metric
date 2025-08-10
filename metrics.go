// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

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

// Registerer registers metrics collectors
type Registerer interface {
	// Register registers a new collector
	Register(Collector) error
	// MustRegister registers a new collector and panics if it fails
	MustRegister(Collector)
	// Unregister unregisters a collector
	Unregister(Collector) bool
}

// Gatherer gathers metrics from all registered collectors
type Gatherer interface {
	// Gather gathers metrics from all registered collectors
	Gather() ([]*MetricFamily, error)
}


// Registry is both a Registerer and a Gatherer
type Registry interface {
	Registerer
	Gatherer
}

// Collector collects metrics
type Collector interface {
	// Describe sends all descriptors of metrics to the channel
	Describe(chan<- *Desc)
	// Collect sends all metrics to the channel
	Collect(chan<- Metric)
}

// Metric represents a single metric value
type Metric interface {
	// Desc returns the descriptor for this metric
	Desc() *Desc
	// Write writes the metric to the given writer
	Write(*MetricDTO) error
}

// Desc describes a metric
type Desc struct {
	// FQName is the fully qualified name of the metric
	FQName string
	// Help is the help text for the metric
	Help string
	// ConstLabels are labels that never change
	ConstLabels Labels
	// VariableLabels are the names of labels that can change
	VariableLabels []string
}

// MetricDTO is a data transfer object for metrics
type MetricDTO struct {
	// Name is the metric name
	Name string
	// Help is the help text
	Help string
	// Type is the metric type
	Type MetricType
	// Value is the metric value
	Value float64
	// Labels are the metric labels
	Labels Labels
	// Timestamp is when the metric was collected
	Timestamp time.Time
}

// MetricType represents the type of a metric
type MetricType int

const (
	// CounterType is a counter metric
	CounterType MetricType = iota
	// GaugeType is a gauge metric
	GaugeType
	// HistogramType is a histogram metric
	HistogramType
	// SummaryType is a summary metric
	SummaryType
)

// MetricFamily is a collection of metrics with the same name
type MetricFamily struct {
	// Name is the metric family name
	Name string
	// Help is the help text
	Help string
	// Type is the metric type
	Type MetricType
	// Metrics are the individual metrics
	Metrics []Metric
}

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
	
	// PrometheusRegistry returns a prometheus-compatible registerer
	PrometheusRegistry() prometheus.Registerer
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

// HTTPHandler handles HTTP requests for metrics
type HTTPHandler interface {
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
var defaultFactory Factory = NewNoOpFactory()

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