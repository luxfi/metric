// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package otel

import (
	"testing"

	"github.com/luxfi/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestToFamilySum checks that a monotonic sum arrives as a counter carrying its
// attributes as label pairs, because that pairing is what the o11y query plane
// reads.
func TestToFamilySum(t *testing.T) {
	fam := toFamily(metricdata.Metrics{
		Name:        "requests_total",
		Description: "Total requests",
		Data: metricdata.Sum[int64]{
			DataPoints: []metricdata.DataPoint[int64]{{
				Attributes: attribute.NewSet(attribute.String("route", "/v1/health")),
				Value:      7,
			}},
		},
	})
	if fam == nil {
		t.Fatal("sum converted to nil family")
	}
	if fam.Name != "requests_total" || fam.Help != "Total requests" {
		t.Errorf("name/help = %q/%q", fam.Name, fam.Help)
	}
	if fam.Type != metric.MetricTypeCounter {
		t.Errorf("type = %v, want counter", fam.Type)
	}
	if len(fam.Metrics) != 1 || fam.Metrics[0].Value.Value != 7 {
		t.Fatalf("metrics = %+v", fam.Metrics)
	}
	want := []metric.LabelPair{{Name: "route", Value: "/v1/health"}}
	if got := fam.Metrics[0].Labels; len(got) != 1 || got[0] != want[0] {
		t.Errorf("labels = %+v, want %+v", got, want)
	}
}

// TestToFamilyGauge checks the float gauge path, the one instrument whose value
// is read straight through with no conversion.
func TestToFamilyGauge(t *testing.T) {
	fam := toFamily(metricdata.Metrics{
		Name: "queue_depth",
		Data: metricdata.Gauge[float64]{
			DataPoints: []metricdata.DataPoint[float64]{{
				Attributes: *attribute.EmptySet(),
				Value:      2.5,
			}},
		},
	})
	if fam == nil || fam.Type != metric.MetricTypeGauge {
		t.Fatalf("family = %+v, want a gauge", fam)
	}
	if fam.Metrics[0].Value.Value != 2.5 {
		t.Errorf("value = %v, want 2.5", fam.Metrics[0].Value.Value)
	}
}

// TestToFamilyHistogramBucketsAccumulate pins the one piece of arithmetic in
// this package: OpenTelemetry reports a count per bucket, Prometheus reports a
// running total, so the counts must be summed as the bounds are walked.
func TestToFamilyHistogramBucketsAccumulate(t *testing.T) {
	fam := toFamily(metricdata.Metrics{
		Name: "request_seconds",
		Data: metricdata.Histogram[float64]{
			DataPoints: []metricdata.HistogramDataPoint[float64]{{
				Attributes:   *attribute.EmptySet(),
				Count:        9,
				Sum:          4.5,
				Bounds:       []float64{0.1, 0.5, 1},
				BucketCounts: []uint64{2, 3, 4, 0},
			}},
		},
	})
	if fam == nil || fam.Type != metric.MetricTypeHistogram {
		t.Fatalf("family = %+v, want a histogram", fam)
	}
	pt := fam.Metrics[0]
	if pt.Value.SampleCount != 9 || pt.Value.SampleSum != 4.5 {
		t.Errorf("count/sum = %v/%v, want 9/4.5", pt.Value.SampleCount, pt.Value.SampleSum)
	}
	want := []metric.Bucket{
		{UpperBound: 0.1, CumulativeCount: 2},
		{UpperBound: 0.5, CumulativeCount: 5},
		{UpperBound: 1, CumulativeCount: 9},
	}
	if len(pt.Value.Buckets) != len(want) {
		t.Fatalf("buckets = %+v, want %+v", pt.Value.Buckets, want)
	}
	for i, b := range pt.Value.Buckets {
		if b != want[i] {
			t.Errorf("bucket %d = %+v, want %+v", i, b, want[i])
		}
	}
}

// TestToFamilyUnsupported checks that an aggregation these shapes cannot carry
// is dropped rather than converted to an empty family, so it never reaches the
// query plane looking like a metric that reported nothing.
func TestToFamilyUnsupported(t *testing.T) {
	fam := toFamily(metricdata.Metrics{
		Name: "request_seconds",
		Data: metricdata.ExponentialHistogram[float64]{
			DataPoints: []metricdata.ExponentialHistogramDataPoint[float64]{{Count: 1}},
		},
	})
	if fam != nil {
		t.Errorf("family = %+v, want nil", fam)
	}
}
