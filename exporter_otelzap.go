// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// OTel metric SDK -> ZAP.
//
// The ZAP exporter speaks MetricFamily, the shape a Prometheus gatherer yields.
// A service instrumented with the OTel metric SDK produces ResourceMetrics
// instead. This adapter is the seam between the two, so such a service can ship
// metrics over ZAP without an OTLP exporter — and therefore without protobuf or
// gRPC, which otlpmetrichttp pulls in as surely as otlpmetricgrpc does.
//
// Histograms decompose into classic bucket/count/sum samples rather than a
// sketch: the ZAP wire carries Prometheus shapes, and the o11y query plane
// already reads them.

package metric

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// OTelZAPExporter adapts the OTel metric SDK to the ZAP wire.
type OTelZAPExporter struct {
	zap *ZAPExporter
}

var _ sdkmetric.Exporter = (*OTelZAPExporter)(nil)

// NewOTelZAPExporter returns an sdkmetric.Exporter that ships over ZAP.
//
// Install it on a periodic reader exactly as an OTLP exporter would be:
//
//	exp, err := metric.NewOTelZAPExporter(metric.ZAPExporterConfig{
//	    Endpoint: "o11y.hanzo.svc:4317", AppName: "ingress",
//	})
//	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
func NewOTelZAPExporter(cfg ZAPExporterConfig) (*OTelZAPExporter, error) {
	z, err := NewZAPExporter(cfg)
	if err != nil {
		return nil, err
	}
	return &OTelZAPExporter{zap: z}, nil
}

// Temporality is cumulative for every instrument: the ZAP wire carries
// Prometheus shapes, and a delta sum has no meaning to a Prometheus reader.
func (e *OTelZAPExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// Aggregation follows the SDK default, which yields explicit-bucket histograms
// — the only histogram this wire can express.
func (e *OTelZAPExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *OTelZAPExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	families := make([]*MetricFamily, 0, 16)
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
func (e *OTelZAPExporter) ForceFlush(context.Context) error { return nil }

func (e *OTelZAPExporter) Shutdown(ctx context.Context) error { return e.zap.Shutdown(ctx) }

// toFamily converts one OTel metric. Unsupported aggregations return nil rather
// than an empty family, so a metric this wire cannot carry is dropped visibly at
// one place instead of arriving empty.
func toFamily(m metricdata.Metrics) *MetricFamily {
	switch d := m.Data.(type) {
	case metricdata.Sum[int64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeCounter,
			Metrics: numberPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Sum[float64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeCounter,
			Metrics: numberPoints(d.DataPoints, func(v float64) float64 { return v })}
	case metricdata.Gauge[int64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeGauge,
			Metrics: numberPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Gauge[float64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeGauge,
			Metrics: numberPoints(d.DataPoints, func(v float64) float64 { return v })}
	case metricdata.Histogram[int64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeHistogram,
			Metrics: histogramPoints(d.DataPoints, func(v int64) float64 { return float64(v) })}
	case metricdata.Histogram[float64]:
		return &MetricFamily{Name: m.Name, Help: m.Description, Type: MetricTypeHistogram,
			Metrics: histogramPoints(d.DataPoints, func(v float64) float64 { return v })}
	default:
		return nil
	}
}

func numberPoints[N int64 | float64](pts []metricdata.DataPoint[N], f func(N) float64) []Metric {
	out := make([]Metric, 0, len(pts))
	for _, p := range pts {
		out = append(out, Metric{Labels: labels(p.Attributes), Value: MetricValue{Value: f(p.Value)}})
	}
	return out
}

func histogramPoints[N int64 | float64](pts []metricdata.HistogramDataPoint[N], f func(N) float64) []Metric {
	out := make([]Metric, 0, len(pts))
	for _, p := range pts {
		// Prometheus buckets are cumulative; OTel counts are per-bucket.
		buckets := make([]Bucket, 0, len(p.Bounds))
		var running uint64
		for i, bound := range p.Bounds {
			if i < len(p.BucketCounts) {
				running += p.BucketCounts[i]
			}
			buckets = append(buckets, Bucket{UpperBound: bound, CumulativeCount: running})
		}
		out = append(out, Metric{
			Labels: labels(p.Attributes),
			Value: MetricValue{
				SampleCount: p.Count,
				SampleSum:   f(p.Sum),
				Buckets:     buckets,
			},
		})
	}
	return out
}

func labels(set attribute.Set) []LabelPair {
	out := make([]LabelPair, 0, set.Len())
	for it := set.Iter(); it.Next(); {
		kv := it.Attribute()
		out = append(out, LabelPair{Name: string(kv.Key), Value: fmt.Sprint(kv.Value.AsInterface())})
	}
	return out
}
