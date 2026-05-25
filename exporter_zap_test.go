// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build metrics

package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/luxfi/zap"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// TestZAPExporter_RoundTrip pins the wire contract:
//
//  1. NewZAPExporter dials a ZAP server.
//  2. ExportGatherer ships a MetricBatch wrapped in a ZAP envelope
//     tagged MsgMetricBatch.
//  3. The server-side handler decodes the JSON payload and recovers
//     every metric family + sample + label byte-for-byte.
//
// No OTel, no OTLP, no Prometheus remote-write. Just ZAP.
func TestZAPExporter_RoundTrip(t *testing.T) {
	port := freePort(t)
	srv := zap.NewNode(zap.NodeConfig{
		NodeID:      "test-o11y",
		ServiceType: "_o11y._tcp",
		Port:        port,
		NoDiscovery: true,
	})

	var (
		mu       sync.Mutex
		received MetricBatch
		gotMsg   = make(chan struct{}, 1)
	)
	srv.Handle(MsgMetricBatch, func(_ context.Context, _ string, m *zap.Message) (*zap.Message, error) {
		payload := append([]byte(nil), m.Root().Bytes(0)...)
		mu.Lock()
		defer mu.Unlock()
		if err := json.Unmarshal(payload, &received); err != nil {
			t.Errorf("unmarshal: %v", err)
			return nil, err
		}
		select {
		case gotMsg <- struct{}{}:
		default:
		}
		return nil, nil
	})

	if err := srv.Start(); err != nil {
		t.Fatalf("server start: %v", err)
	}
	defer srv.Stop()

	// Build a small registry with one of each metric type.
	reg := NewRegistry()
	m := NewWithRegistry("test", reg)
	c := m.NewCounter("requests_total", "Total requests")
	c.Inc()
	c.Add(4)

	g := m.NewGauge("inflight", "Inflight requests")
	g.Set(3)

	h := m.NewHistogram("latency_seconds", "Request latency", DefBuckets)
	h.Observe(0.1)
	h.Observe(0.5)

	// Build the exporter.
	exp, err := NewZAPExporter(ZAPExporterConfig{
		Endpoint: fmt.Sprintf("127.0.0.1:%d", port),
		AppName:  "metric-zap-test",
		Version:  "v0",
		Resource: map[string]string{"env": "test"},
	})
	if err != nil {
		t.Fatalf("new exporter: %v", err)
	}
	defer exp.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := exp.ExportGatherer(ctx, reg); err != nil {
		t.Fatalf("export: %v", err)
	}

	// Wait for the server to receive + decode.
	select {
	case <-gotMsg:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for batch")
	}

	mu.Lock()
	defer mu.Unlock()

	if received.AppName != "metric-zap-test" {
		t.Errorf("appName: got %q want %q", received.AppName, "metric-zap-test")
	}
	if received.Resource["env"] != "test" {
		t.Errorf("resource[env]: got %q want %q", received.Resource["env"], "test")
	}
	if len(received.Families) < 3 {
		t.Fatalf("families: got %d want >=3", len(received.Families))
	}

	// Index families by name for assertions.
	byName := make(map[string]MetricFamilyWire)
	for _, f := range received.Families {
		byName[f.Name] = f
	}

	if cnt, ok := byName["test_requests_total"]; ok {
		if cnt.Type != "counter" {
			t.Errorf("counter type: got %q want counter", cnt.Type)
		}
		if len(cnt.Metrics) == 0 || cnt.Metrics[0].Value == nil || *cnt.Metrics[0].Value != 5 {
			t.Errorf("counter value: got %+v want 5", cnt.Metrics)
		}
	} else {
		t.Errorf("counter family missing: %+v", byName)
	}

	if gauge, ok := byName["test_inflight"]; ok {
		if gauge.Type != "gauge" {
			t.Errorf("gauge type: got %q want gauge", gauge.Type)
		}
		if len(gauge.Metrics) == 0 || gauge.Metrics[0].Value == nil || *gauge.Metrics[0].Value != 3 {
			t.Errorf("gauge value: got %+v want 3", gauge.Metrics)
		}
	} else {
		t.Errorf("gauge family missing: %+v", byName)
	}

	if hist, ok := byName["test_latency_seconds"]; ok {
		if hist.Type != "histogram" {
			t.Errorf("histogram type: got %q want histogram", hist.Type)
		}
		if len(hist.Metrics) == 0 || hist.Metrics[0].SampleCount == nil || *hist.Metrics[0].SampleCount != 2 {
			t.Errorf("histogram sampleCount: got %+v want 2", hist.Metrics)
		}
		if len(hist.Metrics) == 0 || len(hist.Metrics[0].Buckets) == 0 {
			t.Errorf("histogram buckets empty: %+v", hist.Metrics)
		}
	} else {
		t.Errorf("histogram family missing: %+v", byName)
	}
}

// TestZAPExporter_EmptyBatch is a no-op pin: exporting an empty family
// list MUST NOT touch the network.
func TestZAPExporter_EmptyBatch(t *testing.T) {
	exp, err := NewZAPExporter(ZAPExporterConfig{
		Endpoint: "127.0.0.1:1", // unroutable, would error if dialed
		AppName:  "noop",
	})
	if err != nil {
		t.Fatalf("new exporter: %v", err)
	}
	defer exp.Shutdown(context.Background())

	if err := exp.Export(context.Background(), nil); err != nil {
		t.Errorf("empty export must be no-op, got %v", err)
	}
}

// TestZAPExporter_MsgTypeStable pins the wire ID. Changing this number
// is a wire-incompatible break — every deployed collector breaks in
// lockstep. Append-only; never renumber.
func TestZAPExporter_MsgTypeStable(t *testing.T) {
	if MsgMetricBatch != 2 {
		t.Fatalf("MsgMetricBatch wire ID drift: got %d want 2", MsgMetricBatch)
	}
}
