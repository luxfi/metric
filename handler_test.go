// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHandlerForContext(t *testing.T) {
	// Create a context-aware registry
	reg := NewContextRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	reg.MustRegister(counter)
	counter.(interface{ Add(float64) }).Add(10)

	// Test HandlerForContext
	opts := HandlerOpts{
		Timeout: 5 * time.Second,
	}
	handler := HandlerForContext(reg, opts)

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

func TestHandler(t *testing.T) {
	// Create a registry with a metric
	reg := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "Test gauge",
	})
	reg.MustRegister(gauge)
	gauge.Set(42)

	// Get the default handler
	handler := Handler()

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandlerFor(t *testing.T) {
	// Create a registry with metrics
	reg := prometheus.NewRegistry()
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "test_histogram",
		Help:    "Test histogram",
		Buckets: prometheus.DefBuckets,
	})
	reg.MustRegister(histogram)
	histogram.Observe(0.5)

	// Get handler (HandlerFor doesn't take options)
	handler := HandlerFor(reg)

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
	if !contains(body, "test_histogram") {
		t.Error("Expected to find test_histogram in response")
	}
}

func TestWithContextFunc(t *testing.T) {
	// Test WithContextFunc option
	ctxFunc := func(req *http.Request) context.Context {
		return context.WithValue(context.Background(), "test", "value")
	}

	opt := WithContextFunc(ctxFunc)
	if opt == nil {
		t.Error("Expected non-nil HandlerOpt")
	}
}

func TestWithTimeout(t *testing.T) {
	// Test WithTimeout option
	opt := WithTimeout(5 * time.Second)
	if opt == nil {
		t.Error("Expected non-nil HandlerOpt")
	}
}

func TestWithErrorLog(t *testing.T) {
	// Test WithErrorLog option
	logger := func(err error) {
		// Log the error (mock implementation)
		_ = err
	}
	opt := WithErrorLog(logger)
	if opt == nil {
		t.Error("Expected non-nil HandlerOpt")
	}
}

func TestWithMaxRequestsInFlight(t *testing.T) {
	// Test WithMaxRequestsInFlight option
	opt := WithMaxRequestsInFlight(10)
	if opt == nil {
		t.Error("Expected non-nil HandlerOpt")
	}
}

func TestInstrumentMetricHandler(t *testing.T) {
	t.Skip("InstrumentMetricHandler requires specific label configuration")
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create a registry for instrumentation metrics
	reg := prometheus.NewRegistry()

	// Instrument the handler (only takes registerer and handler)
	instrumentedHandler := InstrumentMetricHandler(reg, handler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	instrumentedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

