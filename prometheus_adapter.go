// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// PrometheusRegistry wraps our metrics.Registry to implement prometheus.Registerer
type PrometheusRegistry struct {
	registry Registry
	promReg  prometheus.Registerer
}

// NewPrometheusRegistry creates a wrapper that implements prometheus.Registerer
func NewPrometheusRegistry(metricsReg Registry) prometheus.Registerer {
	// If we have a no-op registry, return a no-op prometheus registerer
	if _, ok := metricsReg.(*noopRegistry); ok {
		return &noopPrometheusRegisterer{}
	}
	
	// Otherwise, use the default prometheus registry
	return prometheus.DefaultRegisterer
}

// noopPrometheusRegisterer implements prometheus.Registerer but does nothing
type noopPrometheusRegisterer struct{}

func (n *noopPrometheusRegisterer) Register(c prometheus.Collector) error {
	return nil
}

func (n *noopPrometheusRegisterer) MustRegister(cs ...prometheus.Collector) {
	// No-op
}

func (n *noopPrometheusRegisterer) Unregister(c prometheus.Collector) bool {
	return true
}

// PrometheusCollectorAdapter adapts our Collector interface to prometheus.Collector
type PrometheusCollectorAdapter struct {
	collector Collector
}

// Describe implements prometheus.Collector
func (p *PrometheusCollectorAdapter) Describe(ch chan<- *prometheus.Desc) {
	// Create a channel to receive our descriptors
	ourCh := make(chan *Desc, 100)
	go func() {
		p.collector.Describe(ourCh)
		close(ourCh)
	}()
	
	// Convert our descriptors to prometheus descriptors
	for desc := range ourCh {
		promDesc := prometheus.NewDesc(
			desc.FQName,
			desc.Help,
			desc.VariableLabels,
			prometheus.Labels(desc.ConstLabels),
		)
		ch <- promDesc
	}
}

// Collect implements prometheus.Collector
func (p *PrometheusCollectorAdapter) Collect(ch chan<- prometheus.Metric) {
	// Create a channel to receive our metrics
	ourCh := make(chan Metric, 100)
	go func() {
		p.collector.Collect(ourCh)
		close(ourCh)
	}()
	
	// Convert our metrics to prometheus metrics
	for metric := range ourCh {
		// Create a DTO to get the metric data
		dto := &MetricDTO{}
		if err := metric.Write(dto); err != nil {
			// Skip metrics that fail to write
			continue
		}
		
		// Create a prometheus metric based on type
		var promMetric prometheus.Metric
		switch dto.Type {
		case CounterType:
			promMetric, _ = prometheus.NewConstMetric(
				prometheus.NewDesc(dto.Name, dto.Help, nil, prometheus.Labels(dto.Labels)),
				prometheus.CounterValue,
				dto.Value,
			)
		case GaugeType:
			promMetric, _ = prometheus.NewConstMetric(
				prometheus.NewDesc(dto.Name, dto.Help, nil, prometheus.Labels(dto.Labels)),
				prometheus.GaugeValue,
				dto.Value,
			)
		case HistogramType, SummaryType:
			// For histogram and summary, we'll use gauge for now
			promMetric, _ = prometheus.NewConstMetric(
				prometheus.NewDesc(dto.Name, dto.Help, nil, prometheus.Labels(dto.Labels)),
				prometheus.UntypedValue,
				dto.Value,
			)
		}
		
		if promMetric != nil {
			ch <- promMetric
		}
	}
}

// PrometheusMetricAdapter adapts prometheus.Metric to our Metric interface
type PrometheusMetricAdapter struct {
	metric prometheus.Metric
}

// Desc implements Metric
func (p *PrometheusMetricAdapter) Desc() *Desc {
	promDesc := p.metric.Desc()
	// Note: We can't fully convert back from prometheus.Desc, so we return a minimal version
	return &Desc{
		FQName: promDesc.String(),
		Help:   "",
	}
}

// Write implements Metric
func (p *PrometheusMetricAdapter) Write(mdto *MetricDTO) error {
	// Convert prometheus metric to our DTO
	var promDTO dto.Metric
	if err := p.metric.Write(&promDTO); err != nil {
		return err
	}
	
	// Extract the value based on type
	if promDTO.Counter != nil {
		mdto.Type = CounterType
		mdto.Value = promDTO.Counter.GetValue()
	} else if promDTO.Gauge != nil {
		mdto.Type = GaugeType
		mdto.Value = promDTO.Gauge.GetValue()
	} else if promDTO.Histogram != nil {
		mdto.Type = HistogramType
		mdto.Value = float64(promDTO.Histogram.GetSampleCount())
	} else if promDTO.Summary != nil {
		mdto.Type = SummaryType
		mdto.Value = float64(promDTO.Summary.GetSampleCount())
	}
	
	// Extract labels
	mdto.Labels = make(Labels)
	for _, label := range promDTO.Label {
		mdto.Labels[label.GetName()] = label.GetValue()
	}
	
	return nil
}