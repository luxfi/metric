// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"net/http"
)

// Export types needed by the node.

// ProcessCollectorOpts are options for the process collector
type ProcessCollectorOpts struct {
	Namespace string
	PidFn     func() (int, error)
}

// MetricDesc describes a metric.
type MetricDesc struct {
	Name string
	Help string
	Type MetricType
}

// NewProcessCollector creates a new process collector (no-op for now).
func NewProcessCollector(opts ProcessCollectorOpts) Collector {
	return &processCollector{opts: opts}
}

// NewGoCollector creates a new Go collector (no-op for now). It accepts the
// Prometheus GoCollectorOption surface (e.g. WithGoCollectorRuntimeMetrics) so
// callers migrating from prometheus/client_golang compile unchanged; the options
// are applied to the collector's config and the no-op collector ignores them.
func NewGoCollector(opts ...GoCollectorOption) Collector {
	cfg := goCollectorConfig{}
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return &goCollector{}
}

// MetricFamilies is a slice of metric families.
type MetricFamilies = []*MetricFamily

// NewHTTPHandler creates an HTTP handler for metrics.
func NewHTTPHandler(gatherer Gatherer, opts HandlerOpts) http.Handler {
	return HandlerForWithOpts(gatherer, opts)
}

// InstrumentMetricHandler wraps handler so each scrape is observed on reg.
// It registers promhttp_metric_handler_requests_total (scrape count) and
// promhttp_metric_handler_requests_in_flight (concurrent scrapes) and updates
// them around each call. Mirrors prometheus/client_golang/promhttp.InstrumentMetricHandler.
func InstrumentMetricHandler(reg Registerer, handler http.Handler) http.Handler {
	requests := reg.NewCounter(
		"promhttp_metric_handler_requests_total",
		"Total number of scrapes served.",
	)
	inFlight := reg.NewGauge(
		"promhttp_metric_handler_requests_in_flight",
		"Current number of scrapes being served.",
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight.Inc()
		defer inFlight.Dec()
		requests.Inc()
		handler.ServeHTTP(w, r)
	})
}
