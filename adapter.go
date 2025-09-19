// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// PrometheusAdapter provides compatibility between our metrics interfaces and prometheus types
// This is needed for components that require direct prometheus types (like api/metrics)

// PrometheusCollector is an alias for prometheus.Collector
// Use this when you need to pass collectors to prometheus-specific code
type PrometheusCollector = prometheus.Collector

// PrometheusGatherer is an alias for prometheus.Gatherer
// Use this when you need prometheus gatherer functionality
type PrometheusGatherer = prometheus.Gatherer

// PrometheusRegisterer is an alias for prometheus.Registerer
// Use this when you need prometheus registerer functionality
type PrometheusRegisterer = prometheus.Registerer

// PrometheusMetric is an alias for prometheus.Metric
type PrometheusMetric = prometheus.Metric

// PrometheusDesc is an alias for prometheus.Desc
type PrometheusDesc = prometheus.Desc

// PrometheusLabels is an alias for prometheus.Labels
type PrometheusLabels = prometheus.Labels

// PrometheusValueType is an alias for prometheus.ValueType
type PrometheusValueType = prometheus.ValueType

// Prometheus value types
const (
	CounterValue PrometheusValueType = prometheus.CounterValue
	GaugeValue   PrometheusValueType = prometheus.GaugeValue
	UntypedValue PrometheusValueType = prometheus.UntypedValue
)

// PrometheusMetricFamily is an alias for dto.MetricFamily
type PrometheusMetricFamily = dto.MetricFamily

// ToPrometheusGatherer converts our Registry to a prometheus.Gatherer
// Since Registry is already *prometheus.Registry, just return it
func ToPrometheusGatherer(r Registry) prometheus.Gatherer {
	return r
}

// ToPrometheusRegisterer converts our Registry to a prometheus.Registerer
// Since Registry is already *prometheus.Registry, just return it
func ToPrometheusRegisterer(r Registry) prometheus.Registerer {
	return r
}

// No adapters needed since Registry is already *prometheus.Registry

// WrapPrometheusRegistererWith wraps a prometheus registerer with labels
func WrapPrometheusRegistererWith(labels prometheus.Labels, reg prometheus.Registerer) prometheus.Registerer {
	return prometheus.WrapRegistererWith(labels, reg)
}

// WrapPrometheusRegistererWithPrefix wraps a prometheus registerer with a prefix
func WrapPrometheusRegistererWithPrefix(prefix string, reg prometheus.Registerer) prometheus.Registerer {
	return prometheus.WrapRegistererWithPrefix(prefix, reg)
}

// NewPrometheusDesc creates a new prometheus descriptor
func NewPrometheusDesc(fqName, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(fqName, help, variableLabels, constLabels)
}

// MustNewPrometheusConstMetric creates a new constant metric
func MustNewPrometheusConstMetric(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labelValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, valueType, value, labelValues...)
}
