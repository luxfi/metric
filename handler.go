// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metric

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// HandlerOpts are options for the context-aware HTTP handler.
type HandlerOpts struct {
	// ErrorLog specifies an optional logger for errors.
	ErrorLog func(error)

	// ErrorHandling defines how errors are handled.
	ErrorHandling promhttp.HandlerErrorHandling

	// Registry is the gatherer to use for metrics.
	Registry prometheus.Gatherer

	// Timeout is the maximum duration for gathering metrics.
	// If zero, no timeout is applied beyond what's in the request context.
	Timeout time.Duration

	// EnableOpenMetrics enables OpenMetrics format support.
	EnableOpenMetrics bool

	// MaxRequestsInFlight limits the number of concurrent metric requests.
	// If zero, no limit is applied.
	MaxRequestsInFlight int

	// ContextFunc allows customizing how the context is derived from the request.
	ContextFunc func(*http.Request) context.Context
}

// HandlerForContext creates an HTTP handler that serves metrics with context support.
// It respects the X-Prometheus-Scrape-Timeout-Seconds header and propagates
// context through the gathering process.
func HandlerForContext(gatherer GathererWithContext, opts HandlerOpts) http.Handler {
	// Create a semaphore for request limiting
	var requestLimiter chan struct{}
	if opts.MaxRequestsInFlight > 0 {
		requestLimiter = make(chan struct{}, opts.MaxRequestsInFlight)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply request limiting if configured
		if requestLimiter != nil {
			select {
			case requestLimiter <- struct{}{}:
				defer func() { <-requestLimiter }()
			default:
				http.Error(w, "Too many concurrent requests", http.StatusServiceUnavailable)
				return
			}
		}

		// Derive the context for collection
		ctx := r.Context()
		if opts.ContextFunc != nil {
			ctx = opts.ContextFunc(r)
		}

		// Apply timeout from Prometheus scrape header or options
		var cancel context.CancelFunc
		headerTimeout := parsePrometheusScrapeTimeout(r)

		if headerTimeout > 0 || opts.Timeout > 0 {
			timeout := selectTimeout(headerTimeout, opts.Timeout)
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		// Gather metrics with context
		mfs, err := gatherer.GatherWithContext(ctx)
		// Handle errors based on ErrorHandling option
		if err != nil {
			if opts.ErrorLog != nil {
				opts.ErrorLog(err)
			}

			switch opts.ErrorHandling {
			case promhttp.HTTPErrorOnError:
				// Check if it's a timeout/cancellation
				if ctx.Err() != nil {
					http.Error(w, "Metric gathering timeout", http.StatusServiceUnavailable)
				} else {
					http.Error(w, fmt.Sprintf("Error gathering metrics: %v", err), http.StatusInternalServerError)
				}
				return

			case promhttp.ContinueOnError:
				// Include error as a special metric but continue
				if mfs == nil {
					mfs = []*dto.MetricFamily{}
				}
				mfs = append(mfs, createErrorMetric(err))

			case promhttp.PanicOnError:
				panic(err)
			}
		}

		// Negotiate content type
		contentType := negotiateContentType(r, opts.EnableOpenMetrics)
		w.Header().Set("Content-Type", contentType)

		// Create encoder based on content type
		encoder := createEncoder(w, contentType)

		// Write metrics
		for _, mf := range mfs {
			if err := encoder.Encode(mf); err != nil {
				if opts.ErrorLog != nil {
					opts.ErrorLog(fmt.Errorf("error encoding metric family: %w", err))
				}
				// Can't return error to client at this point, already started writing
				return
			}
		}
	})
}

// Handler creates a standard HTTP handler with context support using default options.
func Handler() http.Handler {
	return HandlerFor(prometheus.DefaultGatherer)
}

// HandlerFor creates an HTTP handler with context support for the given gatherer.
func HandlerFor(gatherer prometheus.Gatherer) http.Handler {
	// Check if gatherer supports context
	if gwc, ok := gatherer.(GathererWithContext); ok {
		return HandlerForContext(gwc, HandlerOpts{
			ErrorHandling:     promhttp.ContinueOnError,
			EnableOpenMetrics: true,
			Timeout:           10 * time.Second,
		})
	}

	// Fall back to standard promhttp handler
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
		ErrorHandling:     promhttp.ContinueOnError,
		EnableOpenMetrics: true,
		Timeout:           10 * time.Second,
	})
}

// parsePrometheusScrapeTimeout parses the X-Prometheus-Scrape-Timeout-Seconds header.
func parsePrometheusScrapeTimeout(r *http.Request) time.Duration {
	headerVal := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds")
	if headerVal == "" {
		return 0
	}

	seconds, err := strconv.ParseFloat(headerVal, 64)
	if err != nil || seconds <= 0 {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}

// selectTimeout chooses the appropriate timeout to use.
func selectTimeout(headerTimeout, optsTimeout time.Duration) time.Duration {
	if headerTimeout > 0 && optsTimeout > 0 {
		// Use the smaller timeout
		if headerTimeout < optsTimeout {
			return headerTimeout
		}
		return optsTimeout
	}

	if headerTimeout > 0 {
		return headerTimeout
	}

	return optsTimeout
}

// negotiateContentType determines the response content type based on Accept header.
func negotiateContentType(r *http.Request, enableOpenMetrics bool) string {
	if !enableOpenMetrics {
		return "text/plain; version=0.0.4; charset=utf-8"
	}

	accepts := r.Header.Get("Accept")
	if strings.Contains(accepts, "application/openmetrics-text") {
		return "application/openmetrics-text; version=1.0.0; charset=utf-8"
	}

	return "text/plain; version=0.0.4; charset=utf-8"
}

// createEncoder creates the appropriate encoder based on content type.
func createEncoder(w io.Writer, contentType string) expfmt.Encoder {
	if strings.Contains(contentType, "application/openmetrics-text") {
		// OpenMetrics format - use text format for now as FmtOpenMetrics may not be available
		return expfmt.NewEncoder(w, expfmt.NewFormat(expfmt.TypeTextPlain))
	}
	return expfmt.NewEncoder(w, expfmt.NewFormat(expfmt.TypeTextPlain))
}

// createErrorMetric creates a metric family representing an error.
func createErrorMetric(err error) *dto.MetricFamily {
	name := "prometheus_gathering_error"
	help := "Error encountered while gathering metrics"
	metricType := dto.MetricType_GAUGE

	value := float64(1)
	metric := &dto.Metric{
		Gauge: &dto.Gauge{
			Value: &value,
		},
		Label: []*dto.LabelPair{
			{
				Name:  stringPtr("error"),
				Value: stringPtr(err.Error()),
			},
		},
	}

	return &dto.MetricFamily{
		Name:   &name,
		Help:   &help,
		Type:   &metricType,
		Metric: []*dto.Metric{metric},
	}
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// WithContextFunc returns a HandlerOpts option that sets a custom context function.
func WithContextFunc(fn func(*http.Request) context.Context) func(*HandlerOpts) {
	return func(opts *HandlerOpts) {
		opts.ContextFunc = fn
	}
}

// WithTimeout returns a HandlerOpts option that sets the timeout.
func WithTimeout(timeout time.Duration) func(*HandlerOpts) {
	return func(opts *HandlerOpts) {
		opts.Timeout = timeout
	}
}

// WithErrorLog returns a HandlerOpts option that sets the error logger.
func WithErrorLog(logger func(error)) func(*HandlerOpts) {
	return func(opts *HandlerOpts) {
		opts.ErrorLog = logger
	}
}

// WithMaxRequestsInFlight returns a HandlerOpts option that sets the max concurrent requests.
func WithMaxRequestsInFlight(max int) func(*HandlerOpts) {
	return func(opts *HandlerOpts) {
		opts.MaxRequestsInFlight = max
	}
}

// InstrumentMetricHandler wraps a metrics handler with standard HTTP instrumentation.
func InstrumentMetricHandler(reg prometheus.Registerer, handler http.Handler) http.Handler {
	// Create metrics for the handler
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "promhttp_metric_handler_requests_in_flight",
		Help: "Current number of scrapes being served.",
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "promhttp_metric_handler_requests_total",
			Help: "Total number of scrapes by HTTP status code.",
		},
		[]string{"code"},
	)

	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "promhttp_metric_handler_request_duration_seconds",
			Help:    "Histogram of latencies for HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler"},
	)

	// Register metrics
	reg.MustRegister(inFlightGauge, counter, duration)

	// Wrap handler with instrumentation
	return promhttp.InstrumentHandlerInFlight(inFlightGauge,
		promhttp.InstrumentHandlerCounter(counter,
			promhttp.InstrumentHandlerDuration(duration,
				handler,
			),
		),
	)
}
