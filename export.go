// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

// Export prometheus types that are needed by the node

// NewRegistry creates a new registry (internal implementation uses Prometheus)
// This is already exported via the var declaration in metric.go

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

// HTTPHandler creates an HTTP handler for metrics
func HTTPHandler(gatherer prometheus.Gatherer, opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(gatherer, opts)
}

// HTTPHandlerOpts are options for the HTTP handler
type HTTPHandlerOpts = promhttp.HandlerOpts

// MetricFamilies is a slice of metric families
type MetricFamilies = []*dto.MetricFamily

// WrapPrometheusRegistry wraps a prometheus registry in our Registry interface
func WrapPrometheusRegistry(promReg *prometheus.Registry) Registry {
	return promReg
}
