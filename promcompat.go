// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

// This file completes the prometheus/client_golang compatibility surface so
// code migrated mechanically from prometheus (e.g. CoreDNS) compiles and
// behaves against luxfi/metric unchanged. Every symbol here mirrors a
// prometheus/client_golang (or client_golang/prometheus/{promauto,collectors})
// export; the backing implementation is luxfi/metric's own registry.

// AlreadyRegisteredError is returned by a registry's Register when a collector
// with the same fully-qualified name is already registered. It mirrors
// prometheus.AlreadyRegisteredError so callers that tolerate re-registration
// (CoreDNS re-runs plugin setup on every `reload`) can type-assert on it and
// continue rather than failing. The Collector fields are populated when
// available; callers typically only test the error's dynamic type.
type AlreadyRegisteredError struct {
	ExistingCollector Collector
	NewCollector      Collector
}

// Error implements error. The message matches prometheus/client_golang's.
func (AlreadyRegisteredError) Error() string {
	return "duplicate metrics collector registration attempted"
}

// MustRegister registers the given collectors on the default registerer and
// panics on error. Mirrors prometheus.MustRegister.
func MustRegister(cs ...Collector) { DefaultRegisterer.MustRegister(cs...) }

// Unregister removes a previously registered collector from the default
// registerer, returning true if it was present. Mirrors prometheus.Unregister.
func Unregister(c Collector) bool { return DefaultRegisterer.Unregister(c) }

// GoRuntimeMetricsRule selects which runtime/metrics metrics the Go collector
// exposes. Mirrors collectors.GoRuntimeMetricsRule. Matcher is the metric-name
// pattern (kept as the source string; the no-op Go collector does not evaluate
// it).
type GoRuntimeMetricsRule struct {
	Matcher string
}

// MetricsAll enables every available runtime/metrics metric. Mirrors
// collectors.MetricsAll.
var MetricsAll = GoRuntimeMetricsRule{Matcher: "/.*"}

// goCollectorConfig accumulates GoCollectorOption settings.
type goCollectorConfig struct {
	runtimeRules []GoRuntimeMetricsRule
}

// GoCollectorOption configures the Go collector. Mirrors
// collectors.GoCollectorOption.
type GoCollectorOption func(*goCollectorConfig)

// WithGoCollectorRuntimeMetrics enables the runtime/metrics metrics matched by
// the given rules. Mirrors collectors.WithGoCollectorRuntimeMetrics.
func WithGoCollectorRuntimeMetrics(rules ...GoRuntimeMetricsRule) GoCollectorOption {
	return func(c *goCollectorConfig) { c.runtimeRules = append(c.runtimeRules, rules...) }
}

// Factory registers newly-created metrics on a specific Registerer, mirroring
// promauto.Factory. Obtain one with With.
type AutoFactory struct {
	r Registerer
}

// With returns a Factory that registers every metric it creates on r. Mirrors
// promauto.With. A nil r targets the default registerer.
func With(r Registerer) AutoFactory {
	if r == nil {
		r = DefaultRegisterer
	}
	return AutoFactory{r: r}
}

// NewCounterVec creates a counter vector registered on the factory's registerer.
func (f AutoFactory) NewCounterVec(opts CounterOpts, labelNames []string) CounterVec {
	return f.r.NewCounterVec(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help, labelNames)
}

// NewGaugeVec creates a gauge vector registered on the factory's registerer.
func (f AutoFactory) NewGaugeVec(opts GaugeOpts, labelNames []string) GaugeVec {
	return f.r.NewGaugeVec(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help, labelNames)
}

// NewHistogramVec creates a histogram vector registered on the factory's registerer.
func (f AutoFactory) NewHistogramVec(opts HistogramOpts, labelNames []string) HistogramVec {
	return f.r.NewHistogramVec(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help, labelNames, opts.Buckets)
}

// NewCounter creates a counter registered on the factory's registerer.
func (f AutoFactory) NewCounter(opts CounterOpts) Counter {
	return f.r.NewCounter(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help)
}

// NewGauge creates a gauge registered on the factory's registerer.
func (f AutoFactory) NewGauge(opts GaugeOpts) Gauge {
	return f.r.NewGauge(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help)
}

// NewHistogram creates a histogram registered on the factory's registerer.
func (f AutoFactory) NewHistogram(opts HistogramOpts) Histogram {
	return f.r.NewHistogram(prefixedName(AppendNamespace(opts.Namespace, opts.Subsystem), opts.Name), opts.Help, opts.Buckets)
}
