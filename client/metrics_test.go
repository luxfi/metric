//go:build !grpc

// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package io_metric_client

import (
	"encoding/json"
	"testing"
)

func TestMetricType(t *testing.T) {
	tests := []struct {
		mt   MetricType
		want string
	}{
		{MetricType_COUNTER, "COUNTER"},
		{MetricType_GAUGE, "GAUGE"},
		{MetricType_SUMMARY, "SUMMARY"},
		{MetricType_UNTYPED, "UNTYPED"},
		{MetricType_HISTOGRAM, "HISTOGRAM"},
		{MetricType_GAUGE_HISTOGRAM, "GAUGE_HISTOGRAM"},
	}
	for _, tt := range tests {
		if got := tt.mt.String(); got != tt.want {
			t.Errorf("MetricType.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestLabelPair(t *testing.T) {
	name := "test_name"
	value := "test_value"
	lp := &LabelPair{Name: &name, Value: &value}

	if lp.GetName() != name {
		t.Errorf("GetName() = %v, want %v", lp.GetName(), name)
	}
	if lp.GetValue() != value {
		t.Errorf("GetValue() = %v, want %v", lp.GetValue(), value)
	}

	// Test nil case
	var nilLP *LabelPair
	if nilLP.GetName() != "" {
		t.Errorf("nil LabelPair.GetName() = %v, want empty", nilLP.GetName())
	}
}

func TestCounter(t *testing.T) {
	val := 42.0
	c := &Counter{Value: &val}

	if c.GetValue() != val {
		t.Errorf("GetValue() = %v, want %v", c.GetValue(), val)
	}

	// Test Reset
	c.Reset()
	if c.Value != nil {
		t.Error("Reset() did not clear Value")
	}
}

func TestGauge(t *testing.T) {
	val := 3.14
	g := &Gauge{Value: &val}

	if g.GetValue() != val {
		t.Errorf("GetValue() = %v, want %v", g.GetValue(), val)
	}
}

func TestHistogram(t *testing.T) {
	count := uint64(100)
	sum := 500.0
	upperBound := 10.0
	cumCount := uint64(50)

	h := &Histogram{
		SampleCount: &count,
		SampleSum:   &sum,
		Bucket: []*Bucket{
			{UpperBound: &upperBound, CumulativeCount: &cumCount},
		},
	}

	if h.GetSampleCount() != count {
		t.Errorf("GetSampleCount() = %v, want %v", h.GetSampleCount(), count)
	}
	if h.GetSampleSum() != sum {
		t.Errorf("GetSampleSum() = %v, want %v", h.GetSampleSum(), sum)
	}
	if len(h.GetBucket()) != 1 {
		t.Errorf("GetBucket() length = %v, want 1", len(h.GetBucket()))
	}
}

func TestSummary(t *testing.T) {
	count := uint64(200)
	sum := 1000.0
	q := 0.99
	qVal := 42.0

	s := &Summary{
		SampleCount: &count,
		SampleSum:   &sum,
		Quantile: []*Quantile{
			{Quantile: &q, Value: &qVal},
		},
	}

	if s.GetSampleCount() != count {
		t.Errorf("GetSampleCount() = %v, want %v", s.GetSampleCount(), count)
	}
	if s.GetSampleSum() != sum {
		t.Errorf("GetSampleSum() = %v, want %v", s.GetSampleSum(), sum)
	}
	if len(s.GetQuantile()) != 1 {
		t.Errorf("GetQuantile() length = %v, want 1", len(s.GetQuantile()))
	}
}

func TestMetricFamily(t *testing.T) {
	name := "test_metric"
	help := "A test metric"
	typ := MetricType_COUNTER

	mf := &MetricFamily{
		Name: &name,
		Help: &help,
		Type: &typ,
	}

	if mf.GetName() != name {
		t.Errorf("GetName() = %v, want %v", mf.GetName(), name)
	}
	if mf.GetHelp() != help {
		t.Errorf("GetHelp() = %v, want %v", mf.GetHelp(), help)
	}
	if mf.GetType() != typ {
		t.Errorf("GetType() = %v, want %v", mf.GetType(), typ)
	}
}

func TestJSONSerialization(t *testing.T) {
	name := "test"
	val := 1.0
	typ := MetricType_COUNTER

	mf := &MetricFamily{
		Name: &name,
		Type: &typ,
		Metric: []*Metric{
			{
				Counter: &Counter{Value: &val},
			},
		},
	}

	data, err := json.Marshal(mf)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded MetricFamily
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.GetName() != name {
		t.Errorf("decoded Name = %v, want %v", decoded.GetName(), name)
	}
	if len(decoded.GetMetric()) != 1 {
		t.Fatalf("decoded Metric length = %v, want 1", len(decoded.GetMetric()))
	}
	if decoded.GetMetric()[0].GetCounter().GetValue() != val {
		t.Errorf("decoded counter value = %v, want %v", decoded.GetMetric()[0].GetCounter().GetValue(), val)
	}
}
