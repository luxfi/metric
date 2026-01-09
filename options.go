// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import "github.com/prometheus/client_golang/prometheus"

// Option types are aliases to Prometheus options for compatibility.
type CounterOpts = prometheus.CounterOpts
type GaugeOpts = prometheus.GaugeOpts
type HistogramOpts = prometheus.HistogramOpts
type SummaryOpts = prometheus.SummaryOpts

// NewCounterVec creates a wrapped counter vec from options.
func NewCounterVec(opts CounterOpts, labelNames []string) CounterVec {
	return WrapPrometheusCounterVec(prometheus.NewCounterVec(opts, labelNames))
}

// NewGaugeVec creates a wrapped gauge vec from options.
func NewGaugeVec(opts GaugeOpts, labelNames []string) GaugeVec {
	return WrapPrometheusGaugeVec(prometheus.NewGaugeVec(opts, labelNames))
}

// NewHistogramVec creates a wrapped histogram vec from options.
func NewHistogramVec(opts HistogramOpts, labelNames []string) HistogramVec {
	return WrapPrometheusHistogramVec(prometheus.NewHistogramVec(opts, labelNames))
}

// NewSummaryVec creates a wrapped summary vec from options.
func NewSummaryVec(opts SummaryOpts, labelNames []string) SummaryVec {
	return WrapPrometheusSummaryVec(prometheus.NewSummaryVec(opts, labelNames))
}
