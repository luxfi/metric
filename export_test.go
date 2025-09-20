// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestNewProcessCollector(t *testing.T) {
	collector := NewProcessCollector(collectors.ProcessCollectorOpts{})
	if collector == nil {
		t.Fatal("Expected non-nil process collector")
	}

	// Register to a registry
	reg := prometheus.NewRegistry()
	err := reg.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register process collector: %v", err)
	}
}

func TestNewGoCollector(t *testing.T) {
	collector := NewGoCollector()
	if collector == nil {
		t.Fatal("Expected non-nil go collector")
	}

	// Register to a registry
	reg := prometheus.NewRegistry()
	err := reg.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register go collector: %v", err)
	}
}

func TestHTTPHandler(t *testing.T) {
	// Create a registry with some metrics
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)
	counter.Add(42)

	// Get the HTTP handler
	handler := HTTPHandler(reg, promhttp.HandlerOpts{})

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that metrics are returned
	body := w.Body.String()
	if !contains(body, "test_counter") {
		t.Error("Expected to find test_counter in response")
	}
}

func TestWrapPrometheusRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	wrapped := WrapPrometheusRegistry(reg)

	if wrapped == nil {
		t.Fatal("Expected non-nil wrapped registry")
	}

	// Test that it implements Registry interface
	var _ Registry = wrapped

	// The wrapped registry is actually the prometheus registry itself
	// We can register metrics to it
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "help",
	})
	err := reg.Register(counter)
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}