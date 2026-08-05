// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package otel ships metrics recorded by the OpenTelemetry metric SDK over ZAP.
//
// This adapter lives in its own package because importing it costs the caller
// the entire OpenTelemetry metric SDK and everything beneath it — several dozen
// packages that the registry, the text encoder and the ZAP exporter in the root
// package never touch. Plugins such as github.com/zap-proto/zip import the root
// package for exactly those three things and pass whatever it links on to every
// program they are built into, so the root package stays free of OpenTelemetry
// and a caller that wants this adapter asks for it by name.
//
// The ZAP exporter accepts MetricFamily values, the shape a Prometheus gatherer
// yields. A service instrumented with the OpenTelemetry metric SDK produces
// ResourceMetrics instead, so this package translates one into the other. Such
// a service can then ship over ZAP without an OTLP exporter, and therefore
// without protobuf or gRPC, which otlpmetrichttp pulls in as surely as
// otlpmetricgrpc does.
//
// Histograms decompose into classic bucket/count/sum samples rather than a
// sketch: ZAP carries Prometheus shapes, and the o11y query plane already reads
// them.
package otel

import (
	"context"
	"fmt"

	"github.com/luxfi/metric"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Exporter adapts the OpenTelemetry metric SDK to the ZAP exporter.
type Exporter struct {
	zap *metric.ZAPExporter
}

var _ sdkmetric.Exporter = (*Exporter)(nil)

// New returns an sdkmetric.Exporter that ships over ZAP.
//
// Install it on a periodic reader exactly as an OTLP exporter would be:
//
//	exp, err := otel.New(metric.ZAPExporterConfig{
//	    Endpoint: "o11y.hanzo.svc:4317", AppName: "ingress",
//	})
//	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
func New(cfg metric.ZAPExporterConfig) (*Exporter, error) {
	z, err := metric.NewZAPExporter(cfg)
	if err != nil {
		return nil, err
	}
	return &Exporter{zap: z}, nil
}

// Temporality is cumulative for every instrument: ZAP carries Prometheus
// shapes, and a delta sum has no meaning to a Prometheus reader.
func (e *Exporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// Aggregation follows the SDK default, which yields explicit-bucket histograms
// — the only histogram these shapes can express.
func (e *Exporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *Exporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	families := make([]*metric.MetricFamily, 0, 16)
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if fam := toFamily(m); fam != nil {
				families = append(families, fam)
			}
		}
	}
	if len(families) == 0 {
		return nil
	}
	return e.zap.Export(ctx, families)
}

// ForceFlush has nothing to flush: Export sends synchronously.
func (e *Exporter) ForceFlush(context.Context) error { return nil }

func (e *Exporter) Shutdown(ctx context.Context) error { return e.zap.Shutdown(ctx) }

// toFamily converts one OpenTelemetry metric. Unsupported aggregations return
// nil rather than an empty family, so a metric these shapes cannot carry is
// dropped visibly at one place instead of arriving empty.
func toFamily(m metricdata.Metrics) *metric.MetricFamily {
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeCounter,
			Metrics: numberPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Sum[float64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeCounter,
			Metrics: numberPoints(d.DataPoints, func(v float64) float64 { return v })}
	case metricdata.Gauge[int64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeGauge,
			Metrics: numberPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Gauge[float64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeGauge,
			Metrics: numberPoints(d.DataPoints, func(v float64) float64 { return v })}
	case metricdata.Histogram[int64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeHistogram,
			Metrics: histogramPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Histogram[float64]:
		return &metric.MetricFamily{Name: m.Name, Help: m.Description, Type: metric.MetricTypeHistogram,
			Metrics: histogramPoints(d.DataPoints, func(v float64) float64 { return v })}
	default:
		return nil
	}
}

func numberPoints[N int64 | float64](pts []metricdata.DataPoint[N], f func(N) float64) []metric.Metric {
	out := make([]metric.Metric, 0, len(pts))
	for _, p := range pts {
		out = append(out, metric.Metric{Labels: labels(p.Attributes), Value: metric.MetricValue{Value: f(p.Value)}})
	}
	return out
}

func histogramPoints[N int64 | float64](pts []metricdata.HistogramDataPoint[N], f func(N) float64) []metric.Metric {
	out := make([]metric.Metric, 0, len(pts))
	for _, p := range pts {
		// Prometheus buckets are cumulative; OpenTelemetry counts are per-bucket.
		buckets := make([]metric.Bucket, 0, len(p.Bounds))
		var running uint64
		for i, bound := range p.Bounds {
			if i < len(p.BucketCounts) {
				running += p.BucketCounts[i]
			}
			buckets = append(buckets, metric.Bucket{UpperBound: bound, CumulativeCount: running})
		}
		out = append(out, metric.Metric{
			Labels: labels(p.Attributes),
			Value: metric.MetricValue{
				SampleCount: p.Count,
				SampleSum:   f(p.Sum),
				Buckets:     buckets,
			},
		})
	}
	return out
}

func labels(set attribute.Set) []metric.LabelPair {
	out := make([]metric.LabelPair, 0, set.Len())
	for it := set.Iter(); it.Next(); {
		kv := it.Attribute()
		out = append(out, metric.LabelPair{Name: string(kv.Key), Value: fmt.Sprint(kv.Value.AsInterface())})
	}
	return out
}
