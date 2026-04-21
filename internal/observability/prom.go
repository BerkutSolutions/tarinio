package observability

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*metricFamily
	gauges     map[string]*metricFamily
	histograms map[string]*histogramFamily
}

type metricFamily struct {
	name       string
	help       string
	labelNames []string
	samples    map[string]*sample
}

type histogramFamily struct {
	name       string
	help       string
	labelNames []string
	buckets    []float64
	samples    map[string]*histogramSample
}

type sample struct {
	labels map[string]string
	value  float64
}

type histogramSample struct {
	labels  map[string]string
	count   uint64
	sum     float64
	buckets []uint64
}

type CounterVec struct {
	registry *Registry
	family   *metricFamily
}

type GaugeVec struct {
	registry *Registry
	family   *metricFamily
}

type HistogramVec struct {
	registry *Registry
	family   *histogramFamily
}

func NewRegistry() *Registry {
	return &Registry{
		counters:   map[string]*metricFamily{},
		gauges:     map[string]*metricFamily{},
		histograms: map[string]*histogramFamily{},
	}
}

func (r *Registry) Counter(name, help string, labelNames ...string) *CounterVec {
	r.mu.Lock()
	defer r.mu.Unlock()
	family, ok := r.counters[name]
	if !ok {
		family = &metricFamily{
			name:       sanitizeMetricName(name),
			help:       help,
			labelNames: append([]string(nil), labelNames...),
			samples:    map[string]*sample{},
		}
		r.counters[name] = family
	}
	return &CounterVec{registry: r, family: family}
}

func (r *Registry) Gauge(name, help string, labelNames ...string) *GaugeVec {
	r.mu.Lock()
	defer r.mu.Unlock()
	family, ok := r.gauges[name]
	if !ok {
		family = &metricFamily{
			name:       sanitizeMetricName(name),
			help:       help,
			labelNames: append([]string(nil), labelNames...),
			samples:    map[string]*sample{},
		}
		r.gauges[name] = family
	}
	return &GaugeVec{registry: r, family: family}
}

func (r *Registry) Histogram(name, help string, buckets []float64, labelNames ...string) *HistogramVec {
	r.mu.Lock()
	defer r.mu.Unlock()
	family, ok := r.histograms[name]
	if !ok {
		normalizedBuckets := append([]float64(nil), buckets...)
		sort.Float64s(normalizedBuckets)
		family = &histogramFamily{
			name:       sanitizeMetricName(name),
			help:       help,
			labelNames: append([]string(nil), labelNames...),
			buckets:    normalizedBuckets,
			samples:    map[string]*histogramSample{},
		}
		r.histograms[name] = family
	}
	return &HistogramVec{registry: r, family: family}
}

func (c *CounterVec) Add(labels map[string]string, delta float64) {
	if c == nil || c.family == nil || delta == 0 {
		return
	}
	c.registry.mu.Lock()
	defer c.registry.mu.Unlock()
	item := c.family.sampleFor(labels)
	item.value += delta
}

func (c *CounterVec) Inc(labels map[string]string) {
	c.Add(labels, 1)
}

func (g *GaugeVec) Set(labels map[string]string, value float64) {
	if g == nil || g.family == nil {
		return
	}
	g.registry.mu.Lock()
	defer g.registry.mu.Unlock()
	item := g.family.sampleFor(labels)
	item.value = value
}

func (g *GaugeVec) Add(labels map[string]string, delta float64) {
	if g == nil || g.family == nil || delta == 0 {
		return
	}
	g.registry.mu.Lock()
	defer g.registry.mu.Unlock()
	item := g.family.sampleFor(labels)
	item.value += delta
}

func (h *HistogramVec) Observe(labels map[string]string, value float64) {
	if h == nil || h.family == nil {
		return
	}
	h.registry.mu.Lock()
	defer h.registry.mu.Unlock()
	item := h.family.sampleFor(labels)
	item.count++
	item.sum += value
	for i, boundary := range h.family.buckets {
		if value <= boundary {
			item.buckets[i]++
		}
	}
}

func (r *Registry) WritePrometheus(w io.Writer) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.counters)+len(r.gauges)+len(r.histograms))
	types := map[string]string{}
	for name := range r.counters {
		names = append(names, name)
		types[name] = "counter"
	}
	for name := range r.gauges {
		names = append(names, name)
		types[name] = "gauge"
	}
	for name := range r.histograms {
		names = append(names, name)
		types[name] = "histogram"
	}
	sort.Strings(names)

	for _, name := range names {
		switch types[name] {
		case "counter":
			family := r.counters[name]
			if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", family.name, escapeHelp(family.help), family.name); err != nil {
				return err
			}
			for _, key := range sortedMetricKeys(family.samples) {
				item := family.samples[key]
				if _, err := fmt.Fprintf(w, "%s%s %s\n", family.name, renderLabels(item.labels, family.labelNames), formatFloat(item.value)); err != nil {
					return err
				}
			}
		case "gauge":
			family := r.gauges[name]
			if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", family.name, escapeHelp(family.help), family.name); err != nil {
				return err
			}
			for _, key := range sortedMetricKeys(family.samples) {
				item := family.samples[key]
				if _, err := fmt.Fprintf(w, "%s%s %s\n", family.name, renderLabels(item.labels, family.labelNames), formatFloat(item.value)); err != nil {
					return err
				}
			}
		case "histogram":
			family := r.histograms[name]
			if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", family.name, escapeHelp(family.help), family.name); err != nil {
				return err
			}
			for _, key := range sortedHistogramKeys(family.samples) {
				item := family.samples[key]
				cumulative := uint64(0)
				for i, boundary := range family.buckets {
					cumulative += item.buckets[i]
					labels := cloneLabels(item.labels)
					labels["le"] = formatFloat(boundary)
					if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", family.name, renderLabels(labels, append(family.labelNames, "le")), cumulative); err != nil {
						return err
					}
				}
				labels := cloneLabels(item.labels)
				labels["le"] = "+Inf"
				if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", family.name, renderLabels(labels, append(family.labelNames, "le")), item.count); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s_sum%s %s\n", family.name, renderLabels(item.labels, family.labelNames), formatFloat(item.sum)); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s_count%s %d\n", family.name, renderLabels(item.labels, family.labelNames), item.count); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func DefaultDurationBuckets() []float64 {
	return []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}
}

func (f *metricFamily) sampleFor(labels map[string]string) *sample {
	key, normalized := f.normalizeLabels(labels)
	if item, ok := f.samples[key]; ok {
		return item
	}
	item := &sample{labels: normalized}
	f.samples[key] = item
	return item
}

func (f *histogramFamily) sampleFor(labels map[string]string) *histogramSample {
	key, normalized := normalizeLabels(f.labelNames, labels)
	if item, ok := f.samples[key]; ok {
		return item
	}
	item := &histogramSample{
		labels:  normalized,
		buckets: make([]uint64, len(f.buckets)),
	}
	f.samples[key] = item
	return item
}

func (f *metricFamily) normalizeLabels(labels map[string]string) (string, map[string]string) {
	return normalizeLabels(f.labelNames, labels)
}

func normalizeLabels(labelNames []string, labels map[string]string) (string, map[string]string) {
	if len(labelNames) == 0 {
		return "", map[string]string{}
	}
	normalized := make(map[string]string, len(labelNames))
	parts := make([]string, 0, len(labelNames))
	for _, name := range labelNames {
		value := ""
		if labels != nil {
			value = labels[name]
		}
		normalized[name] = value
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, "\xff"), normalized
}

func renderLabels(labels map[string]string, labelNames []string) string {
	if len(labelNames) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labelNames))
	for _, name := range labelNames {
		parts = append(parts, fmt.Sprintf("%s=%q", sanitizeLabelName(name), labels[name]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func sortedMetricKeys(items map[string]*sample) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedHistogramKeys(items map[string]*histogramSample) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sanitizeMetricName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed_metric"
	}
	replacer := strings.NewReplacer("-", "_", ".", "_", " ", "_", "/", "_")
	return replacer.Replace(name)
}

func sanitizeLabelName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "label"
	}
	replacer := strings.NewReplacer("-", "_", ".", "_", " ", "_", "/", "_")
	return replacer.Replace(name)
}

func escapeHelp(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
}

func formatFloat(value float64) string {
	switch {
	case math.IsInf(value, 1):
		return "+Inf"
	case math.IsInf(value, -1):
		return "-Inf"
	case math.IsNaN(value):
		return "NaN"
	default:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}
	return out
}
