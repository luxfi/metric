// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

// WrapPrometheusCounter wraps a prometheus.Counter to implement our Counter interface
func WrapPrometheusCounter(c prometheus.Counter) Counter {
	return &prometheusCounter{counter: c}
}

// WrapPrometheusGauge wraps a prometheus.Gauge to implement our Gauge interface
func WrapPrometheusGauge(g prometheus.Gauge) Gauge {
	return &prometheusGauge{gauge: g}
}

// WrapPrometheusCounterVec wraps a prometheus.CounterVec to implement our CounterVec interface
func WrapPrometheusCounterVec(cv *prometheus.CounterVec) CounterVec {
	return &prometheusCounterVec{vec: cv}
}

// WrapPrometheusGaugeVec wraps a prometheus.GaugeVec to implement our GaugeVec interface  
func WrapPrometheusGaugeVec(gv *prometheus.GaugeVec) GaugeVec {
	return &prometheusGaugeVec{vec: gv}
}

// NewCounterWithOpts creates a wrapped counter from options
func NewCounterWithOpts(opts prometheus.CounterOpts) Counter {
	return WrapPrometheusCounter(prometheus.NewCounter(opts))
}

// NewGaugeWithOpts creates a wrapped gauge from options
func NewGaugeWithOpts(opts prometheus.GaugeOpts) Gauge {
	return WrapPrometheusGauge(prometheus.NewGauge(opts))
}

// NewCounterVecWithOpts creates a wrapped counter vec from options
func NewCounterVecWithOpts(opts prometheus.CounterOpts, labelNames []string) CounterVec {
	return WrapPrometheusCounterVec(prometheus.NewCounterVec(opts, labelNames))
}

// NewGaugeVecWithOpts creates a wrapped gauge vec from options
func NewGaugeVecWithOpts(opts prometheus.GaugeOpts, labelNames []string) GaugeVec {
	return WrapPrometheusGaugeVec(prometheus.NewGaugeVec(opts, labelNames))
}

// AsCollector returns a metric as a prometheus.Collector for registration
func AsCollector(m interface{}) prometheus.Collector {
	// If it already implements Collector, return it
	if c, ok := m.(prometheus.Collector); ok {
		return c
	}
	
	// Otherwise wrap it in a collector adapter
	switch v := m.(type) {
	case Counter:
		return &collectorAdapter{metric: v}
	case Gauge:
		return &collectorAdapter{metric: v}
	case CounterVec:
		return &collectorVecAdapter{vec: v}
	case GaugeVec:
		return &collectorVecAdapter{vec: v}
	default:
		return nil
	}
}

type collectorAdapter struct {
	metric interface{}
}

func (c *collectorAdapter) Describe(ch chan<- *prometheus.Desc) {
	// No-op for compatibility
}

func (c *collectorAdapter) Collect(ch chan<- prometheus.Metric) {
	// No-op for compatibility
}

type collectorVecAdapter struct {
	vec interface{}
}

func (c *collectorVecAdapter) Describe(ch chan<- *prometheus.Desc) {
	// No-op for compatibility
}

func (c *collectorVecAdapter) Collect(ch chan<- prometheus.Metric) {
	// No-op for compatibility
}