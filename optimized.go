// Package metric provides high-performance optimized metrics collection
// Designed for high-performance with minimal allocations and no Prometheus dependency
package metric

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// DefBuckets defines default histogram buckets similar to VictoriaMetrics
var DefBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// OptimizedCounter provides a high-performance counter
// Uses atomic operations to avoid locking overhead
type OptimizedCounter struct {
	value uint64
	name  string
	help  string
}

// NewOptimizedCounter creates a new optimized counter
func NewOptimizedCounter(name, help string) *OptimizedCounter {
	return &OptimizedCounter{
		name: name,
		help: help,
	}
}

// Inc increments the counter by 1
func (c *OptimizedCounter) Inc() {
	atomic.AddUint64(&c.value, 1)
}

// Add adds a value to the counter
func (c *OptimizedCounter) Add(val float64) {
	atomic.AddUint64(&c.value, uint64(val))
}

// Value returns the current value
func (c *OptimizedCounter) Value() uint64 {
	return atomic.LoadUint64(&c.value)
}

// Get returns the current value
func (c *OptimizedCounter) Get() float64 {
	return float64(atomic.LoadUint64(&c.value))
}

// OptimizedGauge provides a high-performance gauge
type OptimizedGauge struct {
	value int64 // Use int64 to handle negative values
	name  string
	help  string
}

// NewOptimizedGauge creates a new optimized gauge
func NewOptimizedGauge(name, help string) *OptimizedGauge {
	return &OptimizedGauge{
		name: name,
		help: help,
	}
}

// Set sets the gauge value
func (g *OptimizedGauge) Set(val float64) {
	atomic.StoreInt64(&g.value, int64(math.Float64bits(val)))
}

// Get returns the gauge value
func (g *OptimizedGauge) Get() float64 {
	return math.Float64frombits(uint64(atomic.LoadInt64(&g.value)))
}

// Inc increments the gauge by 1
func (g *OptimizedGauge) Inc() {
	for {
		oldVal := g.Get()
		newVal := oldVal + 1
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&g.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Dec decrements the gauge by 1
func (g *OptimizedGauge) Dec() {
	for {
		oldVal := g.Get()
		newVal := oldVal - 1
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&g.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Add adds a value to the gauge
func (g *OptimizedGauge) Add(val float64) {
	for {
		oldVal := g.Get()
		newVal := oldVal + val
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&g.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Sub subtracts a value from the gauge
func (g *OptimizedGauge) Sub(val float64) {
	for {
		oldVal := g.Get()
		newVal := oldVal - val
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&g.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Value returns the current value
func (g *OptimizedGauge) Value() float64 {
	return g.Get()
}


// OptimizedHistogram provides a high-performance histogram
// Uses bucket optimization similar to VictoriaMetrics
type OptimizedHistogram struct {
	name        string
	help        string
	buckets     []float64
	bucketCounts []uint64 // Count of values in each bucket
	count       uint64     // Total count of observations
	sum         float64    // Sum of all observations
	mu          sync.RWMutex
}

// NewOptimizedHistogram creates a new optimized histogram
func NewOptimizedHistogram(name, help string, buckets []float64) *OptimizedHistogram {
	// Sort buckets to ensure they're in ascending order
	sortedBuckets := make([]float64, len(buckets))
	copy(sortedBuckets, buckets)
	for i := 0; i < len(sortedBuckets)-1; i++ {
		for j := i + 1; j < len(sortedBuckets); j++ {
			if sortedBuckets[i] > sortedBuckets[j] {
				sortedBuckets[i], sortedBuckets[j] = sortedBuckets[j], sortedBuckets[i]
			}
		}
	}

	return &OptimizedHistogram{
		name:        name,
		help:        help,
		buckets:     sortedBuckets,
		bucketCounts: make([]uint64, len(sortedBuckets)+1), // +1 for +Inf bucket
	}
}

// Observe records a value in the histogram
func (h *OptimizedHistogram) Observe(val float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find the appropriate bucket
	bucketIdx := len(h.buckets) // Default to +Inf bucket
	for i, bucket := range h.buckets {
		if val <= bucket {
			bucketIdx = i
			break
		}
	}

	// Increment the appropriate bucket count
	atomic.AddUint64(&h.bucketCounts[bucketIdx], 1)

	// Increment total count
	atomic.AddUint64(&h.count, 1)

	// Add to sum
	for {
		oldSum := h.sum
		newSum := oldSum + val
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(&h.sum)), math.Float64bits(oldSum), math.Float64bits(newSum)) {
			break
		}
	}
}

// GetBucketCounts returns the current bucket counts
func (h *OptimizedHistogram) GetBucketCounts() []uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]uint64, len(h.bucketCounts))
	for i := range h.bucketCounts {
		result[i] = atomic.LoadUint64(&h.bucketCounts[i])
	}
	return result
}

// GetCount returns the total count
func (h *OptimizedHistogram) GetCount() uint64 {
	return atomic.LoadUint64(&h.count)
}

// GetSum returns the sum
func (h *OptimizedHistogram) GetSum() float64 {
	return h.sum
}

// OptimizedSummary provides a high-performance summary
type OptimizedSummary struct {
	name      string
	help      string
	count     uint64
	sum       float64
	quantiles map[float64]float64 // Quantile -> value
	mu        sync.RWMutex
}

// NewOptimizedSummary creates a new optimized summary
func NewOptimizedSummary(name, help string) *OptimizedSummary {
	return &OptimizedSummary{
		name:      name,
		help:      help,
		quantiles: make(map[float64]float64),
	}
}

// Observe records a value in the summary
func (s *OptimizedSummary) Observe(val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddUint64(&s.count, 1)

	// Add to sum atomically
	for {
		oldSum := s.sum
		newSum := oldSum + val
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(&s.sum)), math.Float64bits(oldSum), math.Float64bits(newSum)) {
			break
		}
	}

	// Note: In a real VictoriaMetrics implementation, quantiles would be calculated differently
	// This is a simplified version for demonstration purposes
}

// GetCount returns the total count
func (s *OptimizedSummary) GetCount() uint64 {
	return atomic.LoadUint64(&s.count)
}

// GetSum returns the sum
func (s *OptimizedSummary) GetSum() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sum
}

// MetricsRegistry provides a high-performance metrics registry
type MetricsRegistry struct {
	counters   map[string]*OptimizedCounter
	gauges     map[string]*OptimizedGauge
	histograms map[string]*OptimizedHistogram
	summaries  map[string]*OptimizedSummary
	mu         sync.RWMutex
}

// NewMetricsRegistry creates a new metrics registry
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		counters:   make(map[string]*OptimizedCounter),
		gauges:     make(map[string]*OptimizedGauge),
		histograms: make(map[string]*OptimizedHistogram),
		summaries:  make(map[string]*OptimizedSummary),
	}
}

// RegisterCounter registers a counter
func (r *MetricsRegistry) RegisterCounter(name string, counter *OptimizedCounter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] = counter
}

// GetCounter gets a counter by name
func (r *MetricsRegistry) GetCounter(name string) *OptimizedCounter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counters[name]
}

// RegisterGauge registers a gauge
func (r *MetricsRegistry) RegisterGauge(name string, gauge *OptimizedGauge) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = gauge
}

// GetGauge gets a gauge by name
func (r *MetricsRegistry) GetGauge(name string) *OptimizedGauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.gauges[name]
}

// RegisterHistogram registers a histogram
func (r *MetricsRegistry) RegisterHistogram(name string, histogram *OptimizedHistogram) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.histograms[name] = histogram
}

// GetHistogram gets a histogram by name
func (r *MetricsRegistry) GetHistogram(name string) *OptimizedHistogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.histograms[name]
}

// RegisterSummary registers a summary
func (r *MetricsRegistry) RegisterSummary(name string, summary *OptimizedSummary) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summaries[name] = summary
}

// GetSummary gets a summary by name
func (r *MetricsRegistry) GetSummary(name string) *OptimizedSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.summaries[name]
}

// GetMetrics returns all metrics in a format similar to Prometheus exposition format
func (r *MetricsRegistry) GetMetrics() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder

	// Add all counters
	for name, counter := range r.counters {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, counter.help, name, name, counter.Value()))
	}

	// Add all gauges
	for name, gauge := range r.gauges {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s gauge\n%s %f\n", name, gauge.help, name, name, gauge.Value()))
	}

	// Add all histograms
	for name, histogram := range r.histograms {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s histogram\n", name, histogram.help, name))

		// Write bucket counts
		cumulative := uint64(0)
		for i, bucket := range histogram.buckets {
			cumulative += atomic.LoadUint64(&histogram.bucketCounts[i])
			sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%g\"} %d\n", name, bucket, cumulative))
		}

		// Write +Inf bucket
		cumulative += atomic.LoadUint64(&histogram.bucketCounts[len(histogram.buckets)])
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", name, cumulative))

		// Write count and sum
		sb.WriteString(fmt.Sprintf("%s_count %d\n", name, atomic.LoadUint64(&histogram.count)))
		sb.WriteString(fmt.Sprintf("%s_sum %g\n", name, histogram.sum))
		sb.WriteString("\n")
	}

	// Add all summaries
	for name, summary := range r.summaries {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s summary\n", name, summary.help, name))

		// Write count and sum
		sb.WriteString(fmt.Sprintf("%s_count %d\n", name, atomic.LoadUint64(&summary.count)))
		sb.WriteString(fmt.Sprintf("%s_sum %g\n", name, summary.sum))

		// Write quantiles (simplified - in real implementation, quantiles would be calculated properly)
		for quantile, value := range summary.quantiles {
			sb.WriteString(fmt.Sprintf("%s{quantile=\"%g\"} %g\n", name, quantile, value))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// TimingMetric provides timing functionality similar to VictoriaMetrics
type TimingMetric struct {
	histogram *OptimizedHistogram
	start     time.Time
}

// NewTimingMetric creates a new timing metric
func NewTimingMetric(histogram *OptimizedHistogram) *TimingMetric {
	return &TimingMetric{
		histogram: histogram,
		start:     time.Now(),
	}
}

// Stop stops the timing and records the duration
func (t *TimingMetric) Stop() {
	duration := time.Since(t.start).Seconds()
	t.histogram.Observe(duration)
}

// Reset resets the timing
func (t *TimingMetric) Reset() {
	t.start = time.Now()
}

// Duration returns the current duration
func (t *TimingMetric) Duration() time.Duration {
	return time.Since(t.start)
}