// Package metric provides high-performance metrics collection
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

// VictoriaCounter provides a high-performance counter without Prometheus dependency
type VictoriaCounter struct {
	value uint64
	name  string
	help  string
}

// NewVictoriaCounter creates a new VictoriaMetrics-style counter
func NewVictoriaCounter(name, help string) *VictoriaCounter {
	return &VictoriaCounter{
		name: name,
		help: help,
	}
}

// Inc increments the counter by 1
func (vc *VictoriaCounter) Inc() {
	atomic.AddUint64(&vc.value, 1)
}

// Add adds a value to the counter
func (vc *VictoriaCounter) Add(val float64) {
	atomic.AddUint64(&vc.value, uint64(val))
}

// Value returns the current value
func (vc *VictoriaCounter) Value() uint64 {
	return atomic.LoadUint64(&vc.value)
}

// Get returns the current value as float64.
func (vc *VictoriaCounter) Get() float64 {
	return float64(atomic.LoadUint64(&vc.value))
}

// String returns the counter in Prometheus exposition format
func (vc *VictoriaCounter) String() string {
	return fmt.Sprintf("# HELP %s %s\n# TYPE %s counter\n%s %d", vc.name, vc.help, vc.name, vc.name, vc.Value())
}

// VictoriaGauge provides a high-performance gauge without Prometheus dependency
type VictoriaGauge struct {
	value int64 // Use int64 to handle negative values
	name  string
	help  string
}

// NewVictoriaGauge creates a new VictoriaMetrics-style gauge
func NewVictoriaGauge(name, help string) *VictoriaGauge {
	return &VictoriaGauge{
		name: name,
		help: help,
	}
}

// Set sets the gauge value
func (vg *VictoriaGauge) Set(val float64) {
	atomic.StoreInt64(&vg.value, int64(math.Float64bits(val)))
}

// Get returns the gauge value
func (vg *VictoriaGauge) Get() float64 {
	return math.Float64frombits(uint64(atomic.LoadInt64(&vg.value)))
}

// Inc increments the gauge by 1
func (vg *VictoriaGauge) Inc() {
	for {
		oldVal := vg.Get()
		newVal := oldVal + 1
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&vg.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Dec decrements the gauge by 1
func (vg *VictoriaGauge) Dec() {
	for {
		oldVal := vg.Get()
		newVal := oldVal - 1
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&vg.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Add adds a value to the gauge
func (vg *VictoriaGauge) Add(val float64) {
	for {
		oldVal := vg.Get()
		newVal := oldVal + val
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&vg.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// Sub subtracts a value from the gauge
func (vg *VictoriaGauge) Sub(val float64) {
	for {
		oldVal := vg.Get()
		newVal := oldVal - val
		oldBits := math.Float64bits(oldVal)
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapInt64(&vg.value, int64(oldBits), int64(newBits)) {
			break
		}
	}
}

// String returns the gauge in Prometheus exposition format
func (vg *VictoriaGauge) String() string {
	return fmt.Sprintf("# HELP %s %s\n# TYPE %s gauge\n%s %f", vg.name, vg.help, vg.name, vg.name, vg.Get())
}

// Value returns the current value.
func (vg *VictoriaGauge) Value() float64 {
	return vg.Get()
}

// VictoriaHistogram provides a high-performance histogram without Prometheus dependency
type VictoriaHistogram struct {
	name        string
	help        string
	buckets     []float64
	bucketCounts []uint64 // Count of values in each bucket
	count       uint64     // Total count of observations
	sum         float64    // Sum of all observations
	mu          sync.RWMutex
}

// NewVictoriaHistogram creates a new VictoriaMetrics-style histogram
func NewVictoriaHistogram(name, help string, buckets []float64) *VictoriaHistogram {
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

	return &VictoriaHistogram{
		name:        name,
		help:        help,
		buckets:     sortedBuckets,
		bucketCounts: make([]uint64, len(sortedBuckets)+1), // +1 for +Inf bucket
	}
}

// Observe records a value in the histogram
func (vh *VictoriaHistogram) Observe(val float64) {
	vh.mu.Lock()
	defer vh.mu.Unlock()

	// Find the appropriate bucket
	bucketIdx := len(vh.buckets) // Default to +Inf bucket
	for i, bucket := range vh.buckets {
		if val <= bucket {
			bucketIdx = i
			break
		}
	}

	// Increment the appropriate bucket count
	atomic.AddUint64(&vh.bucketCounts[bucketIdx], 1)
	
	// Increment total count
	atomic.AddUint64(&vh.count, 1)
	
	// Add to sum
	for {
		oldSum := vh.sum
		newSum := oldSum + val
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(&vh.sum)), math.Float64bits(oldSum), math.Float64bits(newSum)) {
			break
		}
	}
}

// GetBucketCounts returns the current bucket counts
func (vh *VictoriaHistogram) GetBucketCounts() []uint64 {
	vh.mu.RLock()
	defer vh.mu.RUnlock()
	
	result := make([]uint64, len(vh.bucketCounts))
	for i := range vh.bucketCounts {
		result[i] = atomic.LoadUint64(&vh.bucketCounts[i])
	}
	return result
}

// GetCount returns the total count
func (vh *VictoriaHistogram) GetCount() uint64 {
	return atomic.LoadUint64(&vh.count)
}

// GetSum returns the sum
func (vh *VictoriaHistogram) GetSum() float64 {
	return vh.sum
}

// String returns the histogram in Prometheus exposition format
func (vh *VictoriaHistogram) String() string {
	vh.mu.RLock()
	defer vh.mu.RUnlock()
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", vh.name, vh.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", vh.name))
	
	// Write bucket counts
	cumulative := uint64(0)
	for i, bucket := range vh.buckets {
		cumulative += atomic.LoadUint64(&vh.bucketCounts[i])
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%g\"} %d\n", vh.name, bucket, cumulative))
	}
	
	// Write +Inf bucket
	cumulative += atomic.LoadUint64(&vh.bucketCounts[len(vh.buckets)])
	sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", vh.name, cumulative))
	
	// Write count and sum
	sb.WriteString(fmt.Sprintf("%s_count %d\n", vh.name, atomic.LoadUint64(&vh.count)))
	sb.WriteString(fmt.Sprintf("%s_sum %g\n", vh.name, vh.sum))
	
	return sb.String()
}

// VictoriaSummary provides a high-performance summary without Prometheus dependency
type VictoriaSummary struct {
	name      string
	help      string
	count     uint64
	sum       float64
	quantiles map[float64]float64 // Quantile -> value
	mu        sync.RWMutex
}

// NewVictoriaSummary creates a new VictoriaMetrics-style summary
func NewVictoriaSummary(name, help string) *VictoriaSummary {
	return &VictoriaSummary{
		name:      name,
		help:      help,
		quantiles: make(map[float64]float64),
	}
}

// Observe records a value in the summary
func (vs *VictoriaSummary) Observe(val float64) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	atomic.AddUint64(&vs.count, 1)
	
	// Add to sum atomically
	for {
		oldSum := vs.sum
		newSum := oldSum + val
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(&vs.sum)), math.Float64bits(oldSum), math.Float64bits(newSum)) {
			break
		}
	}
	
	// Note: In a real VictoriaMetrics implementation, quantiles would be calculated differently
	// This is a simplified version for demonstration purposes
}

// GetCount returns the total count
func (vs *VictoriaSummary) GetCount() uint64 {
	return atomic.LoadUint64(&vs.count)
}

// GetSum returns the sum
func (vs *VictoriaSummary) GetSum() float64 {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.sum
}

// String returns the summary in Prometheus exposition format
func (vs *VictoriaSummary) String() string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", vs.name, vs.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s summary\n", vs.name))
	
	// Write count and sum
	sb.WriteString(fmt.Sprintf("%s_count %d\n", vs.name, atomic.LoadUint64(&vs.count)))
	sb.WriteString(fmt.Sprintf("%s_sum %g\n", vs.name, vs.sum))
	
	// Write quantiles (simplified - in real implementation, quantiles would be calculated properly)
	for quantile, value := range vs.quantiles {
		sb.WriteString(fmt.Sprintf("%s{quantile=\"%g\"} %g\n", vs.name, quantile, value))
	}
	
	return sb.String()
}

// HighPerfMetricsRegistry provides a registry for high-performance metrics
type HighPerfMetricsRegistry struct {
	counters   map[string]*VictoriaCounter
	gauges     map[string]*VictoriaGauge
	histograms map[string]*VictoriaHistogram
	summaries  map[string]*VictoriaSummary
	mu         sync.RWMutex
}

// NewHighPerfMetricsRegistry creates a new high-performance registry
func NewHighPerfMetricsRegistry() *HighPerfMetricsRegistry {
	return &HighPerfMetricsRegistry{
		counters:   make(map[string]*VictoriaCounter),
		gauges:     make(map[string]*VictoriaGauge),
		histograms: make(map[string]*VictoriaHistogram),
		summaries:  make(map[string]*VictoriaSummary),
	}
}

// RegisterCounter registers a counter
func (hpr *HighPerfMetricsRegistry) RegisterCounter(name string, counter *VictoriaCounter) {
	hpr.mu.Lock()
	defer hpr.mu.Unlock()
	hpr.counters[name] = counter
}

// RegisterGauge registers a gauge
func (hpr *HighPerfMetricsRegistry) RegisterGauge(name string, gauge *VictoriaGauge) {
	hpr.mu.Lock()
	defer hpr.mu.Unlock()
	hpr.gauges[name] = gauge
}

// RegisterHistogram registers a histogram
func (hpr *HighPerfMetricsRegistry) RegisterHistogram(name string, histogram *VictoriaHistogram) {
	hpr.mu.Lock()
	defer hpr.mu.Unlock()
	hpr.histograms[name] = histogram
}

// RegisterSummary registers a summary
func (hpr *HighPerfMetricsRegistry) RegisterSummary(name string, summary *VictoriaSummary) {
	hpr.mu.Lock()
	defer hpr.mu.Unlock()
	hpr.summaries[name] = summary
}

// GetMetrics returns all metrics in Prometheus exposition format
func (hpr *HighPerfMetricsRegistry) GetMetrics() string {
	hpr.mu.RLock()
	defer hpr.mu.RUnlock()

	var sb strings.Builder

	// Add all counters
	for _, counter := range hpr.counters {
		sb.WriteString(counter.String())
		sb.WriteString("\n")
	}

	// Add all gauges
	for _, gauge := range hpr.gauges {
		sb.WriteString(gauge.String())
		sb.WriteString("\n")
	}

	// Add all histograms
	for _, histogram := range hpr.histograms {
		sb.WriteString(histogram.String())
		sb.WriteString("\n")
	}

	// Add all summaries
	for _, summary := range hpr.summaries {
		sb.WriteString(summary.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

// VictoriaTimingMetric provides timing functionality similar to VictoriaMetrics
type VictoriaTimingMetric struct {
	histogram *VictoriaHistogram
	start     time.Time
}

// NewVictoriaTimingMetric creates a new timing metric
func NewVictoriaTimingMetric(histogram *VictoriaHistogram) *VictoriaTimingMetric {
	return &VictoriaTimingMetric{
		histogram: histogram,
		start:     time.Now(),
	}
}

// Stop stops the timing and records the duration
func (vtm *VictoriaTimingMetric) Stop() {
	duration := time.Since(vtm.start).Seconds()
	vtm.histogram.Observe(duration)
}

// Reset resets the timing
func (vtm *VictoriaTimingMetric) Reset() {
	vtm.start = time.Now()
}

// Duration returns the current duration
func (vtm *VictoriaTimingMetric) Duration() time.Duration {
	return time.Since(vtm.start)
}

// Start starts the timer and returns a function to stop it
func (vtm *VictoriaTimingMetric) Start() func() {
	vtm.start = time.Now()
	return vtm.Stop
}

// ObserveTime observes the given duration
func (vtm *VictoriaTimingMetric) ObserveTime(d time.Duration) {
	vtm.histogram.Observe(d.Seconds())
}

// HighPerfMetricsFactory creates high-performance metrics
type HighPerfMetricsFactory struct {
	registry *VictoriaMetricsRegistry
}

// NewHighPerfMetricsFactory creates a factory that produces high-performance metrics
func NewHighPerfMetricsFactory() *HighPerfMetricsFactory {
	return &HighPerfMetricsFactory{
		registry: NewVictoriaMetricsRegistry(),
	}
}

// New creates a new metrics instance with the given namespace.
func (hpf *HighPerfMetricsFactory) New(namespace string) Metrics {
	return &highPerfMetrics{
		namespace: namespace,
		factory:   hpf,
	}
}

// NewWithRegistry creates a new metrics instance, ignoring the registry for high-perf metrics.
func (hpf *HighPerfMetricsFactory) NewWithRegistry(namespace string, _ Registry) Metrics {
	return &highPerfMetrics{
		namespace: namespace,
		factory:   hpf,
	}
}

// NewCounter creates a new high-performance counter
func (hpf *HighPerfMetricsFactory) NewCounter(name, help string) Counter {
	counter := NewVictoriaCounter(name, help)
	hpf.registry.RegisterCounter(name, counter)
	return counter
}

// NewGauge creates a new high-performance gauge
func (hpf *HighPerfMetricsFactory) NewGauge(name, help string) Gauge {
	gauge := NewVictoriaGauge(name, help)
	hpf.registry.RegisterGauge(name, gauge)
	return gauge
}

// NewHistogram creates a new high-performance histogram
func (hpf *HighPerfMetricsFactory) NewHistogram(name, help string, buckets []float64) Histogram {
	histogram := NewVictoriaHistogram(name, help, buckets)
	hpf.registry.RegisterHistogram(name, histogram)
	return histogram
}

// NewSummary creates a new high-performance summary
func (hpf *HighPerfMetricsFactory) NewSummary(name, help string, _ map[float64]float64) Summary {
	summary := NewVictoriaSummary(name, help)
	hpf.registry.RegisterSummary(name, summary)
	return summary
}

// GetRegistry returns the underlying registry
func (hpf *HighPerfMetricsFactory) GetRegistry() *VictoriaMetricsRegistry {
	return hpf.registry
}

type highPerfMetrics struct {
	namespace string
	factory   *HighPerfMetricsFactory
}

func (m *highPerfMetrics) NewCounter(name, help string) Counter {
	return m.factory.NewCounter(prefixedName(m.namespace, name), help)
}

func (m *highPerfMetrics) NewCounterVec(name, help string, labelNames []string) CounterVec {
	return newHighPerfCounterVec(m.factory, prefixedName(m.namespace, name), help, labelNames)
}

func (m *highPerfMetrics) NewGauge(name, help string) Gauge {
	return m.factory.NewGauge(prefixedName(m.namespace, name), help)
}

func (m *highPerfMetrics) NewGaugeVec(name, help string, labelNames []string) GaugeVec {
	return newHighPerfGaugeVec(m.factory, prefixedName(m.namespace, name), help, labelNames)
}

func (m *highPerfMetrics) NewHistogram(name, help string, buckets []float64) Histogram {
	return m.factory.NewHistogram(prefixedName(m.namespace, name), help, buckets)
}

func (m *highPerfMetrics) NewHistogramVec(name, help string, labelNames []string, buckets []float64) HistogramVec {
	return newHighPerfHistogramVec(m.factory, prefixedName(m.namespace, name), help, labelNames, buckets)
}

func (m *highPerfMetrics) NewSummary(name, help string, objectives map[float64]float64) Summary {
	return m.factory.NewSummary(prefixedName(m.namespace, name), help, objectives)
}

func (m *highPerfMetrics) NewSummaryVec(name, help string, labelNames []string, objectives map[float64]float64) SummaryVec {
	return newHighPerfSummaryVec(m.factory, prefixedName(m.namespace, name), help, labelNames, objectives)
}

func (m *highPerfMetrics) Registry() Registry {
	return nil
}

func (m *highPerfMetrics) PrometheusRegistry() interface{} {
	return nil
}

func prefixedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "_" + name
}

type highPerfCounterVec struct {
	factory    *HighPerfMetricsFactory
	name       string
	help       string
	labelNames []string
	mu         sync.Mutex
	counters   map[string]Counter
}

func newHighPerfCounterVec(factory *HighPerfMetricsFactory, name, help string, labelNames []string) *highPerfCounterVec {
	return &highPerfCounterVec{
		factory:    factory,
		name:       name,
		help:       help,
		labelNames: append([]string(nil), labelNames...),
		counters:   make(map[string]Counter),
	}
}

func (v *highPerfCounterVec) With(labels Labels) Counter {
	key := labelsKey(v.labelNames, labels)
	return v.getOrCreate(key)
}

func (v *highPerfCounterVec) WithLabelValues(values ...string) Counter {
	key := valuesKey(v.labelNames, values)
	return v.getOrCreate(key)
}

func (v *highPerfCounterVec) getOrCreate(key string) Counter {
	v.mu.Lock()
	defer v.mu.Unlock()
	if c, ok := v.counters[key]; ok {
		return c
	}
	counter := v.factory.NewCounter(v.name+key, v.help)
	v.counters[key] = counter
	return counter
}

type highPerfGaugeVec struct {
	factory    *HighPerfMetricsFactory
	name       string
	help       string
	labelNames []string
	mu         sync.Mutex
	gauges     map[string]Gauge
}

func newHighPerfGaugeVec(factory *HighPerfMetricsFactory, name, help string, labelNames []string) *highPerfGaugeVec {
	return &highPerfGaugeVec{
		factory:    factory,
		name:       name,
		help:       help,
		labelNames: append([]string(nil), labelNames...),
		gauges:     make(map[string]Gauge),
	}
}

func (v *highPerfGaugeVec) With(labels Labels) Gauge {
	key := labelsKey(v.labelNames, labels)
	return v.getOrCreate(key)
}

func (v *highPerfGaugeVec) WithLabelValues(values ...string) Gauge {
	key := valuesKey(v.labelNames, values)
	return v.getOrCreate(key)
}

func (v *highPerfGaugeVec) getOrCreate(key string) Gauge {
	v.mu.Lock()
	defer v.mu.Unlock()
	if g, ok := v.gauges[key]; ok {
		return g
	}
	gauge := v.factory.NewGauge(v.name+key, v.help)
	v.gauges[key] = gauge
	return gauge
}

type highPerfHistogramVec struct {
	factory    *HighPerfMetricsFactory
	name       string
	help       string
	labelNames []string
	buckets    []float64
	mu         sync.Mutex
	histograms map[string]Histogram
}

func newHighPerfHistogramVec(factory *HighPerfMetricsFactory, name, help string, labelNames []string, buckets []float64) *highPerfHistogramVec {
	return &highPerfHistogramVec{
		factory:    factory,
		name:       name,
		help:       help,
		labelNames: append([]string(nil), labelNames...),
		buckets:    append([]float64(nil), buckets...),
		histograms: make(map[string]Histogram),
	}
}

func (v *highPerfHistogramVec) With(labels Labels) Histogram {
	key := labelsKey(v.labelNames, labels)
	return v.getOrCreate(key)
}

func (v *highPerfHistogramVec) WithLabelValues(values ...string) Histogram {
	key := valuesKey(v.labelNames, values)
	return v.getOrCreate(key)
}

func (v *highPerfHistogramVec) getOrCreate(key string) Histogram {
	v.mu.Lock()
	defer v.mu.Unlock()
	if h, ok := v.histograms[key]; ok {
		return h
	}
	histogram := v.factory.NewHistogram(v.name+key, v.help, v.buckets)
	v.histograms[key] = histogram
	return histogram
}

type highPerfSummaryVec struct {
	factory    *HighPerfMetricsFactory
	name       string
	help       string
	labelNames []string
	objectives map[float64]float64
	mu         sync.Mutex
	summaries  map[string]Summary
}

func newHighPerfSummaryVec(factory *HighPerfMetricsFactory, name, help string, labelNames []string, objectives map[float64]float64) *highPerfSummaryVec {
	objCopy := make(map[float64]float64, len(objectives))
	for k, v := range objectives {
		objCopy[k] = v
	}
	return &highPerfSummaryVec{
		factory:    factory,
		name:       name,
		help:       help,
		labelNames: append([]string(nil), labelNames...),
		objectives: objCopy,
		summaries:  make(map[string]Summary),
	}
}

func (v *highPerfSummaryVec) With(labels Labels) Summary {
	key := labelsKey(v.labelNames, labels)
	return v.getOrCreate(key)
}

func (v *highPerfSummaryVec) WithLabelValues(values ...string) Summary {
	key := valuesKey(v.labelNames, values)
	return v.getOrCreate(key)
}

func (v *highPerfSummaryVec) getOrCreate(key string) Summary {
	v.mu.Lock()
	defer v.mu.Unlock()
	if s, ok := v.summaries[key]; ok {
		return s
	}
	summary := v.factory.NewSummary(v.name+key, v.help, v.objectives)
	v.summaries[key] = summary
	return summary
}

func labelsKey(labelNames []string, labels Labels) string {
	if len(labelNames) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("{")
	for i, name := range labelNames {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(name)
		sb.WriteString("=\"")
		sb.WriteString(labels[name])
		sb.WriteString("\"")
	}
	sb.WriteString("}")
	return sb.String()
}

func valuesKey(labelNames []string, values []string) string {
	if len(labelNames) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("{")
	for i, name := range labelNames {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(name)
		sb.WriteString("=\"")
		if i < len(values) {
			sb.WriteString(values[i])
		}
		sb.WriteString("\"")
	}
	sb.WriteString("}")
	return sb.String()
}
