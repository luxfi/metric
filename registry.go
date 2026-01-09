// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry aliases the Prometheus registry type.
// We keep a pointer alias to avoid an extra wrapper type.
type Registry = *prometheus.Registry

// Registerer aliases the Prometheus Registerer interface.
type Registerer = prometheus.Registerer

// NewRegistry returns a new Prometheus registry.
func NewRegistry() Registry {
	return prometheus.NewRegistry()
}

// VictoriaMetricsRegistry provides a minimal registry for VictoriaMetrics-style
// metrics without pulling in a heavy dependency.
type VictoriaMetricsRegistry struct {
	mu         sync.Mutex
	counters   map[string]*VictoriaCounter
	gauges     map[string]*VictoriaGauge
	histograms map[string]*VictoriaHistogram
	summaries  map[string]*VictoriaSummary
}

// NewVictoriaMetricsRegistry creates an empty VictoriaMetricsRegistry.
func NewVictoriaMetricsRegistry() *VictoriaMetricsRegistry {
	return &VictoriaMetricsRegistry{
		counters:   make(map[string]*VictoriaCounter),
		gauges:     make(map[string]*VictoriaGauge),
		histograms: make(map[string]*VictoriaHistogram),
		summaries:  make(map[string]*VictoriaSummary),
	}
}

// RegisterCounter records a counter by name, returning the existing one if present.
func (r *VictoriaMetricsRegistry) RegisterCounter(name string, counter *VictoriaCounter) *VictoriaCounter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.counters[name]; ok {
		return existing
	}
	r.counters[name] = counter
	return counter
}

// RegisterGauge records a gauge by name, returning the existing one if present.
func (r *VictoriaMetricsRegistry) RegisterGauge(name string, gauge *VictoriaGauge) *VictoriaGauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.gauges[name]; ok {
		return existing
	}
	r.gauges[name] = gauge
	return gauge
}

// RegisterHistogram records a histogram by name, returning the existing one if present.
func (r *VictoriaMetricsRegistry) RegisterHistogram(name string, histogram *VictoriaHistogram) *VictoriaHistogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.histograms[name]; ok {
		return existing
	}
	r.histograms[name] = histogram
	return histogram
}

// RegisterSummary records a summary by name, returning the existing one if present.
func (r *VictoriaMetricsRegistry) RegisterSummary(name string, summary *VictoriaSummary) *VictoriaSummary {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.summaries[name]; ok {
		return existing
	}
	r.summaries[name] = summary
	return summary
}
