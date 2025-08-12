# Lux Metrics Library

A comprehensive metrics library for the Lux ecosystem with built-in context propagation support for Prometheus metrics collection.

## Features

- **Full Prometheus Compatibility**: Works seamlessly with Prometheus client libraries
- **Context Propagation**: Pass `context.Context` through the entire metrics collection pipeline
- **Timeout Support**: Respects Prometheus scrape timeouts via `X-Prometheus-Scrape-Timeout-Seconds` header
- **Request-Scoped Metrics**: Filter or customize metrics based on request parameters
- **Backward Compatible**: Existing collectors work without modification
- **Clean Abstraction**: No Prometheus types leak outside the package
- **Flexible Architecture**: Support for multiple gatherers, registries, and custom collectors

## Installation

```bash
go get github.com/luxfi/metric@latest
```

## Quick Start

### Basic Usage

```go
package main

import (
    "net/http"
    metrics "github.com/luxfi/metric"
)

func main() {
    // Create a new metrics instance
    m := metrics.New("myapp")
    
    // Create metrics
    counter := m.NewCounter("requests_total", "Total number of requests")
    gauge := m.NewGauge("temperature_celsius", "Current temperature")
    histogram := m.NewHistogram("request_duration_seconds", "Request duration", 
        []float64{0.1, 0.5, 1, 2, 5})
    
    // Use metrics
    counter.Inc()
    gauge.Set(23.5)
    histogram.Observe(0.234)
    
    // Expose metrics endpoint
    http.Handle("/metrics", metrics.Handler())
    http.ListenAndServe(":8080", nil)
}
```

### Context-Aware Collectors

The library supports context propagation for advanced use cases:

```go
package main

import (
    "context"
    "database/sql"
    "github.com/prometheus/client_golang/prometheus"
    metrics "github.com/luxfi/metric"
)

// Custom collector that respects context cancellation
type DatabaseCollector struct {
    db *sql.DB
}

func (c *DatabaseCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- prometheus.NewDesc("db_connections", "Number of database connections", nil, nil)
}

func (c *DatabaseCollector) Collect(ch chan<- prometheus.Metric) {
    c.CollectWithContext(context.Background(), ch)
}

func (c *DatabaseCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    // Check if context is cancelled
    select {
    case <-ctx.Done():
        return // Stop collection if cancelled
    default:
    }
    
    // Perform expensive operation with context
    var connections float64
    err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pg_stat_activity").Scan(&connections)
    if err != nil {
        return
    }
    
    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("db_connections", "Number of database connections", nil, nil),
        prometheus.GaugeValue,
        connections,
    )
}

func main() {
    // Create context-aware registry
    reg := metrics.NewContextRegistry()
    
    // Register context-aware collector
    dbCollector := &DatabaseCollector{db: getDB()}
    reg.MustRegister(dbCollector)
    
    // Create handler with timeout support
    handler := metrics.HandlerForContext(reg, metrics.HandlerOpts{
        Timeout: 10 * time.Second,
        ErrorHandling: promhttp.ContinueOnError,
        MaxRequestsInFlight: 5,
    })
    
    http.Handle("/metrics", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Request-Scoped Metrics

Filter metrics based on request parameters:

```go
handler := metrics.HandlerForContext(reg, metrics.HandlerOpts{
    ContextFunc: func(r *http.Request) context.Context {
        ctx := r.Context()
        
        // Pass query parameters to collectors
        if includes := r.URL.Query().Get("include"); includes != "" {
            ctx = context.WithValue(ctx, "metrics.include", includes)
        }
        
        return ctx
    },
})

// In your collector:
func (c *MyCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    // Check what metrics to include
    if includes, ok := ctx.Value("metrics.include").(string); ok {
        if !strings.Contains(includes, "expensive") {
            return // Skip expensive metrics
        }
    }
    
    // Collect expensive metrics...
}
```

## API Reference

### Core Interfaces

#### `CollectorWithContext`
```go
type CollectorWithContext interface {
    prometheus.Collector
    CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric)
}
```

#### `GathererWithContext`
```go
type GathererWithContext interface {
    prometheus.Gatherer
    GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error)
}
```

### Registry

#### `ContextRegistry`
A registry that supports both standard and context-aware collectors:

```go
reg := metrics.NewContextRegistry()

// Register standard collector
reg.MustRegister(prometheus.NewGoCollector())

// Register context-aware collector
reg.MustRegister(myContextCollector)

// Gather with context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
metrics, err := reg.GatherWithContext(ctx)
```

### HTTP Handler

#### `HandlerForContext`
Creates an HTTP handler with full context support:

```go
handler := metrics.HandlerForContext(gatherer, metrics.HandlerOpts{
    Timeout:             10 * time.Second,
    ErrorHandling:       promhttp.ContinueOnError,
    MaxRequestsInFlight: 10,
    EnableOpenMetrics:   true,
    ContextFunc: func(r *http.Request) context.Context {
        // Custom context derivation
        return r.Context()
    },
})
```

### Utilities

#### `CollectorFunc`
Adapter for using functions as collectors:

```go
collector := metrics.NewCollectorFunc(
    func(ch chan<- *prometheus.Desc) {
        ch <- prometheus.NewDesc("my_metric", "help", nil, nil)
    },
    func(ctx context.Context, ch chan<- prometheus.Metric) {
        // Collect with context
        select {
        case <-ctx.Done():
            return
        default:
            // Collect metrics...
        }
    },
)
```

#### `CollectorAdapter`
Wraps standard collectors to be context-aware:

```go
standardCollector := prometheus.NewGoCollector()
contextAware := metrics.NewCollectorAdapter(standardCollector)
```

## Advanced Features

### Multi-Gatherer Support

Combine metrics from multiple sources with namespace prefixes:

```go
mg := metrics.NewMultiGathererWithContext()

// Register different registries with namespaces
mg.Register("app", appRegistry)
mg.Register("database", dbRegistry)
mg.Register("cache", cacheRegistry)

// All metrics will be prefixed with their namespace
// e.g., app_requests_total, database_connections, cache_hits
```

### Timeout Handling

The library automatically respects Prometheus scrape timeouts:

1. Reads `X-Prometheus-Scrape-Timeout-Seconds` header from Prometheus
2. Applies the timeout to the context
3. Context-aware collectors can check `ctx.Done()` to stop early
4. Returns partial results if timeout occurs

### Error Handling

Configure how errors are handled during metric collection:

```go
handler := metrics.HandlerForContext(reg, metrics.HandlerOpts{
    ErrorHandling: promhttp.HTTPErrorOnError,  // Return HTTP error
    // or
    ErrorHandling: promhttp.ContinueOnError,   // Include error metric
    // or  
    ErrorHandling: promhttp.PanicOnError,      // Panic on error
    
    ErrorLog: func(err error) {
        log.Printf("Metrics error: %v", err)
    },
})
```

## Best Practices

1. **Always check context in long-running collectors**:
   ```go
   select {
   case <-ctx.Done():
       return // Stop if cancelled
   default:
       // Continue collection
   }
   ```

2. **Use context for expensive operations**:
   ```go
   rows, err := db.QueryContext(ctx, query)
   ```

3. **Set reasonable timeouts**:
   ```go
   HandlerOpts{
       Timeout: 10 * time.Second, // Default timeout
   }
   ```

4. **Limit concurrent requests**:
   ```go
   HandlerOpts{
       MaxRequestsInFlight: 10, // Prevent overload
   }
   ```

5. **Log errors for debugging**:
   ```go
   HandlerOpts{
       ErrorLog: log.Printf,
   }
   ```

## Migration Guide

### From Standard Prometheus

Existing Prometheus collectors work without modification:

```go
// Before
reg := prometheus.NewRegistry()
reg.MustRegister(collector)
http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

// After - with context support
reg := metrics.NewContextRegistry()
reg.MustRegister(collector) // Same collector works
http.Handle("/metrics", metrics.HandlerFor(reg))
```

### Adding Context Support

To make a collector context-aware:

```go
// Before
func (c *MyCollector) Collect(ch chan<- prometheus.Metric) {
    expensiveOperation()
    ch <- metric
}

// After
func (c *MyCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    select {
    case <-ctx.Done():
        return // Respect cancellation
    default:
    }
    
    expensiveOperationWithContext(ctx)
    ch <- metric
}

// Keep the old method for compatibility
func (c *MyCollector) Collect(ch chan<- prometheus.Metric) {
    c.CollectWithContext(context.Background(), ch)
}
```

## Performance Considerations

- Context-aware collectors run concurrently by default
- Cancelled contexts stop collection immediately
- Non-context collectors continue to completion
- Use `MaxRequestsInFlight` to limit concurrent scrapes
- Timeout applies to entire gathering operation

## Testing

Run the test suite:

```bash
go test ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass
2. New features include tests
3. Documentation is updated
4. Code follows Go best practices

## License

Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.

See the file LICENSE for licensing terms.