package collector

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
	MetricTypeUntyped   MetricType = "untyped"
)

type Label struct {
	Name  string
	Value string
}

func LabelsFromMap(m map[string]string) []Label {
	if len(m) == 0 {
		return nil
	}
	out := make([]Label, 0, len(m))
	for k, v := range m {
		out = append(out, Label{Name: k, Value: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func FormatLabels(labels []Label) string {
	if len(labels) == 0 {
		return ""
	}
	var b strings.Builder
	for i, l := range labels {
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(l.Name)
		b.WriteString(`="`)
		// Keep console output readable; we don't need full Prometheus escaping here.
		b.WriteString(l.Value)
		b.WriteString(`"`)
	}
	return b.String()
}

type Sample struct {
	Labels    []Label
	Value     float64
	Timestamp *time.Time
}

type Histogram struct {
	Labels    []Label
	Buckets   []Bucket // ordered by UpperBound
	Count     uint64
	Sum       float64
	Timestamp *time.Time
}

type Bucket struct {
	UpperBound float64
	Count      uint64
}

type Summary struct {
	Labels     []Label
	Quantiles  []Quantile // ordered by Quantile
	Count      uint64
	Sum        float64
	Timestamp  *time.Time
}

type Quantile struct {
	Quantile float64
	Value    float64
}

type MetricFamily struct {
	Name string
	Help string
	Type MetricType

	Samples    []Sample
	Histograms []Histogram
	Summaries  []Summary
}

type Collector interface {
	Name() string
	Describe() string
	Collect(ctx context.Context) ([]MetricFamily, error)
}

type CollectorStatus struct {
	Name        string
	Description string
	Implemented bool
}

func (mf MetricFamily) SamplesCount() int {
	switch mf.Type {
	case MetricTypeHistogram:
		// Each histogram expands to multiple series (_bucket, _sum, _count) in Prometheus exposition,
		// but for table purposes we count one logical histogram per labelset.
		return len(mf.Histograms)
	case MetricTypeSummary:
		return len(mf.Summaries)
	default:
		return len(mf.Samples)
	}
}

func (mf MetricFamily) Example() (labels string, value string) {
	switch mf.Type {
	case MetricTypeHistogram:
		if len(mf.Histograms) == 0 {
			return "", ""
		}
		h := mf.Histograms[0]
		return FormatLabels(h.Labels), "histogram"
	case MetricTypeSummary:
		if len(mf.Summaries) == 0 {
			return "", ""
		}
		s := mf.Summaries[0]
		return FormatLabels(s.Labels), "summary"
	default:
		if len(mf.Samples) == 0 {
			return "", ""
		}
		s := mf.Samples[0]
		return FormatLabels(s.Labels), strconv.FormatFloat(s.Value, 'f', -1, 64)
	}
}

