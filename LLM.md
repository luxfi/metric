# LLM Documentation - Lux Metric Library

## Overview

The `github.com/luxfi/metric` library is a comprehensive metrics package for the Lux ecosystem that provides Prometheus-compatible metrics with enhanced context propagation support. This document provides detailed technical information for AI assistants and developers working with this library.

## Architecture

### Core Design Principles

1. **Context Propagation**: The library extends standard Prometheus interfaces to support `context.Context` throughout the metrics collection pipeline
2. **Backward Compatibility**: All existing Prometheus collectors work without modification
3. **Clean Abstraction**: Prometheus types are wrapped and don't leak outside the package
4. **Concurrent Collection**: Collectors run in parallel with proper synchronization
5. **Timeout Awareness**: Respects Prometheus scrape timeouts and client disconnections

### Package Structure

```
metric/
├── metrics.go          # Core interfaces and types
├── prometheus.go       # Prometheus implementation
├── context.go          # Context-aware collectors and registries
├── handler.go          # HTTP handlers with context support
├── gatherer.go         # Multi-gatherer implementations
├── adapter.go          # Type adapters and aliases
├── export.go           # Public exports
├── noop.go            # No-op implementations for testing
├── context_test.go     # Context functionality tests
└── metrics_test.go     # Core metrics tests
```

## Key Components

### Context-Aware Interfaces

#### CollectorWithContext
```go
type CollectorWithContext interface {
    prometheus.Collector  // Embeds base interface
    CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric)
}
```

This interface extends the standard Prometheus collector to accept a context. Implementations should:
- Check `ctx.Done()` periodically for cancellation
- Use context for timeouts on expensive operations
- Pass context to database queries, HTTP requests, etc.
- Fall back gracefully when context is cancelled

#### GathererWithContext
```go
type GathererWithContext interface {
    prometheus.Gatherer  // Embeds base interface
    GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error)
}
```

Gatherers aggregate metrics from multiple collectors. The context-aware version:
- Propagates context to all collectors
- Stops gathering if context is cancelled
- Returns partial results on timeout
- Handles mixed standard/context collectors

### ContextRegistry

The `ContextRegistry` is the core component that manages collectors:

```go
type ContextRegistry struct {
    mu         sync.RWMutex
    collectors map[uint64]collectorEntry
    nextID     uint64
    pedantic   bool  // Extra validation if true
}
```

Key features:
- **Type Detection**: Automatically detects if a collector supports context via type assertion
- **Concurrent Collection**: Spawns goroutines for parallel metric collection
- **Error Handling**: Captures panics and errors from collectors
- **Duplicate Detection**: Optional pedantic mode checks for duplicate descriptors
- **Mixed Mode**: Supports both standard and context-aware collectors simultaneously

### HTTP Handler

The `HandlerForContext` function creates an HTTP handler with full context support:

```go
func HandlerForContext(gatherer GathererWithContext, opts HandlerOpts) http.Handler
```

Features:
- **Timeout Header**: Reads `X-Prometheus-Scrape-Timeout-Seconds` from Prometheus
- **Context Derivation**: Custom `ContextFunc` for request-specific context
- **Request Limiting**: Optional max concurrent requests via semaphore
- **Content Negotiation**: Supports both text and OpenMetrics formats
- **Error Handling**: Configurable error behavior (continue, HTTP error, panic)

### Metric Collection Flow

1. **HTTP Request** arrives at `/metrics` endpoint
2. **Handler** extracts/creates context with timeout
3. **Registry** receives `GatherWithContext(ctx)` call
4. **Parallel Collection**:
   - Each collector runs in its own goroutine
   - Context-aware collectors receive the context
   - Standard collectors use background context
5. **Aggregation**: Metrics are collected into families
6. **Encoding**: Results encoded to Prometheus format
7. **Response**: Sent to client (or error on timeout)

## Implementation Details

### Context Propagation Mechanism

The context flows through the system as follows:

```
HTTP Request
    ↓
HandlerForContext (creates context with timeout)
    ↓
GathererWithContext (propagates to collectors)
    ↓
CollectorWithContext (uses context for operations)
```

### Timeout Handling

Timeouts are handled at multiple levels:

1. **HTTP Level**: Request context timeout from client
2. **Prometheus Header**: `X-Prometheus-Scrape-Timeout-Seconds`
3. **Handler Options**: Configured timeout in `HandlerOpts`
4. **Effective Timeout**: Minimum of all applicable timeouts

### Concurrent Collection

The registry uses a fan-out/fan-in pattern:

```go
// Fan-out: Start collectors
for _, collector := range collectors {
    go func(c Collector) {
        c.Collect(ch)  // or CollectWithContext
    }(collector)
}

// Fan-in: Aggregate results
for metric := range ch {
    families[metric.Desc()] = append(...)
}
```

### Error Recovery

The system handles various error conditions:

- **Panics**: Caught and converted to errors
- **Timeouts**: Return partial results with error
- **Cancellations**: Stop collection immediately
- **Duplicate Metrics**: Detected and reported (in pedantic mode)

## Usage Patterns

### Basic Metrics

Standard metrics work without any changes:

```go
m := metrics.New("app")
counter := m.NewCounter("requests", "Total requests")
counter.Inc()
```

### Context-Aware Collector

For expensive operations that should respect timeouts:

```go
type DBCollector struct {
    db *sql.DB
}

func (c *DBCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    // Quick context check
    if ctx.Err() != nil {
        return
    }
    
    // Use context for query
    var count float64
    err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil {
        return
    }
    
    ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, count)
}
```

### Request-Scoped Metrics

Filter metrics based on request parameters:

```go
HandlerOpts{
    ContextFunc: func(r *http.Request) context.Context {
        ctx := r.Context()
        // Add request info to context
        ctx = context.WithValue(ctx, "include", r.URL.Query().Get("include"))
        return ctx
    },
}
```

### Graceful Degradation

Collectors should handle context cancellation gracefully:

```go
func (c *Collector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    // Start expensive operation
    resultCh := make(chan float64)
    go func() {
        resultCh <- expensiveOperation()
    }()
    
    // Wait with timeout
    select {
    case result := <-resultCh:
        ch <- createMetric(result)
    case <-ctx.Done():
        // Context cancelled, return partial or no results
        return
    }
}
```

## Performance Considerations

### Concurrency

- Collectors run in parallel by default
- Use goroutine pool for many collectors
- Channel buffer size affects memory usage
- Consider collector grouping for related metrics

### Memory Usage

- Metric families are built in memory
- Large numbers of labels increase memory
- Channel buffers should be sized appropriately
- Consider streaming for very large metric sets

### CPU Usage

- Context checking has minimal overhead
- Type assertions are cached per collector
- Parallel collection improves CPU utilization
- Timeout handling prevents runaway collectors

## Testing Strategies

### Unit Tests

Test individual collectors with mock contexts:

```go
func TestCollector(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    ch := make(chan prometheus.Metric, 10)
    collector.CollectWithContext(ctx, ch)
    close(ch)
    
    // Verify metrics
    for metric := range ch {
        // Check metric values
    }
}
```

### Integration Tests

Test the full pipeline with HTTP requests:

```go
func TestHandler(t *testing.T) {
    handler := metrics.HandlerForContext(registry, opts)
    server := httptest.NewServer(handler)
    defer server.Close()
    
    // Add timeout header
    req, _ := http.NewRequest("GET", server.URL, nil)
    req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", "1")
    
    resp, err := http.DefaultClient.Do(req)
    // Verify response
}
```

### Timeout Tests

Verify timeout handling with slow collectors:

```go
func TestTimeout(t *testing.T) {
    reg := metrics.NewContextRegistry()
    reg.MustRegister(&SlowCollector{delay: 5 * time.Second})
    
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    _, err := reg.GatherWithContext(ctx)
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Error("Expected timeout")
    }
}
```

## Migration Guide

### From prometheus/client_golang

1. **Import Change**:
```go
// Before
import "github.com/prometheus/client_golang/prometheus"

// After
import metrics "github.com/luxfi/metric"
```

2. **Registry Creation**:
```go
// Before
reg := prometheus.NewRegistry()

// After (with context support)
reg := metrics.NewContextRegistry()
```

3. **Handler Creation**:
```go
// Before
handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

// After
handler := metrics.HandlerFor(reg)
```

### Adding Context Support to Existing Collectors

1. **Add the interface**:
```go
func (c *MyCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    // Move logic here with context checks
}
```

2. **Delegate old method**:
```go
func (c *MyCollector) Collect(ch chan<- prometheus.Metric) {
    c.CollectWithContext(context.Background(), ch)
}
```

3. **Add context checks**:
```go
select {
case <-ctx.Done():
    return
default:
    // Continue
}
```

## Common Pitfalls

### 1. Not Checking Context

**Wrong**:
```go
func (c *Collector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    time.Sleep(10 * time.Second)  // Ignores context
    ch <- metric
}
```

**Right**:
```go
func (c *Collector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    select {
    case <-time.After(10 * time.Second):
        ch <- metric
    case <-ctx.Done():
        return
    }
}
```

### 2. Blocking Channel Writes

**Wrong**:
```go
ch <- metric  // Can block if buffer full
```

**Right**:
```go
select {
case ch <- metric:
case <-ctx.Done():
    return
}
```

### 3. Not Handling Panics

**Wrong**:
```go
go func() {
    collector.Collect(ch)  // Panic crashes goroutine
}()
```

**Right**:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            errCh <- fmt.Errorf("panic: %v", r)
        }
    }()
    collector.Collect(ch)
}()
```

## Advanced Topics

### Custom Gatherers

Implement `GathererWithContext` for custom aggregation:

```go
type ShardedGatherer struct {
    shards []GathererWithContext
}

func (g *ShardedGatherer) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
    var wg sync.WaitGroup
    results := make([][]*dto.MetricFamily, len(g.shards))
    
    for i, shard := range g.shards {
        wg.Add(1)
        go func(idx int, s GathererWithContext) {
            defer wg.Done()
            results[idx], _ = s.GatherWithContext(ctx)
        }(i, shard)
    }
    
    wg.Wait()
    return mergeResults(results), nil
}
```

### Dynamic Collector Registration

Register/unregister collectors at runtime:

```go
type DynamicRegistry struct {
    *ContextRegistry
    mu sync.RWMutex
}

func (r *DynamicRegistry) RegisterDynamic(name string, c prometheus.Collector) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Store by name for later removal
    r.collectors[name] = c
    r.ContextRegistry.Register(c)
}
```

### Metric Filtering

Filter metrics based on context values:

```go
func (c *FilteredCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    filter, _ := ctx.Value("filter").(string)
    
    for _, metric := range c.allMetrics {
        if matchesFilter(metric, filter) {
            ch <- metric
        }
    }
}
```

## Debugging Tips

### Enable Debug Logging

```go
HandlerOpts{
    ErrorLog: func(err error) {
        log.Printf("[METRICS DEBUG] %v", err)
    },
}
```

### Trace Context Flow

```go
func (c *Collector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
    log.Printf("Collector %s: context deadline: %v", c.name, ctx.Deadline())
    // ...
}
```

### Monitor Goroutines

```go
func (r *Registry) GatherWithContext(ctx context.Context) ([]*dto.MetricFamily, error) {
    before := runtime.NumGoroutine()
    defer func() {
        after := runtime.NumGoroutine()
        log.Printf("Goroutines: %d -> %d", before, after)
    }()
    // ...
}
```

## Future Enhancements

Potential areas for future development:

1. **Streaming Metrics**: Support for streaming large metric sets
2. **Metric Caching**: Cache expensive metrics with TTL
3. **Distributed Collection**: Gather metrics from remote sources
4. **Metric Transformations**: Transform metrics in the pipeline
5. **Observability**: Built-in metrics about metric collection
6. **Circuit Breakers**: Protect against failing collectors
7. **Rate Limiting**: Per-collector rate limits
8. **Metric Priorities**: Collect critical metrics first

## Version History

- **v1.2.2**: Current version with full context propagation support
- **v1.2.1**: Bug fixes and performance improvements
- **v1.2.0**: Added multi-gatherer support
- **v1.1.9**: Initial public release
- **v1.0.0**: Internal version for Lux ecosystem

## References

- [Prometheus Client Library](https://github.com/prometheus/client_golang)
- [Context Package](https://golang.org/pkg/context/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [OpenMetrics Specification](https://openmetrics.io/)

---

*This document is maintained for AI assistants and developers working with the Lux metric library. Last updated: 2025*

## Context for All AI Assistants

This file (`LLM.md`) is symlinked as:
- `.AGENTS.md`
- `CLAUDE.md`
- `QWEN.md`
- `GEMINI.md`

All files reference the same knowledge base. Updates here propagate to all AI systems.

## Rules for AI Assistants

1. **ALWAYS** update LLM.md with significant discoveries
2. **NEVER** commit symlinked files (.AGENTS.md, CLAUDE.md, etc.) - they're in .gitignore
3. **NEVER** create random summary files - update THIS file
