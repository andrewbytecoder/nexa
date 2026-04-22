package collector

import (
	"sort"

	dto "github.com/prometheus/client_model/go"
)

func DTOToNexa(mfs []*dto.MetricFamily) ([]MetricFamily, error) {
	out := make([]MetricFamily, 0, len(mfs))
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		f, err := convertOneDTO(mf)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func convertOneDTO(mf *dto.MetricFamily) (MetricFamily, error) {
	name := mf.GetName()
	help := mf.GetHelp()

	var t MetricType
	switch mf.GetType() {
	case dto.MetricType_COUNTER:
		t = MetricTypeCounter
	case dto.MetricType_GAUGE:
		t = MetricTypeGauge
	case dto.MetricType_UNTYPED:
		t = MetricTypeUntyped
	case dto.MetricType_HISTOGRAM:
		t = MetricTypeHistogram
	case dto.MetricType_SUMMARY:
		t = MetricTypeSummary
	default:
		t = MetricTypeUntyped
	}

	f := MetricFamily{Name: name, Help: help, Type: t}

	switch t {
	case MetricTypeHistogram:
		for _, m := range mf.GetMetric() {
			h := m.GetHistogram()
			if h == nil {
				continue
			}
			buckets := make([]Bucket, 0, len(h.GetBucket()))
			for _, b := range h.GetBucket() {
				buckets = append(buckets, Bucket{UpperBound: b.GetUpperBound(), Count: b.GetCumulativeCount()})
			}
			sort.Slice(buckets, func(i, j int) bool { return buckets[i].UpperBound < buckets[j].UpperBound })
			f.Histograms = append(f.Histograms, Histogram{
				Labels:  convertDTOLabels(m.GetLabel()),
				Buckets: buckets,
				Count:   h.GetSampleCount(),
				Sum:     h.GetSampleSum(),
			})
		}
	case MetricTypeSummary:
		for _, m := range mf.GetMetric() {
			s := m.GetSummary()
			if s == nil {
				continue
			}
			qs := make([]Quantile, 0, len(s.GetQuantile()))
			for _, q := range s.GetQuantile() {
				qs = append(qs, Quantile{Quantile: q.GetQuantile(), Value: q.GetValue()})
			}
			sort.Slice(qs, func(i, j int) bool { return qs[i].Quantile < qs[j].Quantile })
			f.Summaries = append(f.Summaries, Summary{
				Labels:    convertDTOLabels(m.GetLabel()),
				Quantiles: qs,
				Count:     s.GetSampleCount(),
				Sum:       s.GetSampleSum(),
			})
		}
	default:
		for _, m := range mf.GetMetric() {
			v, ok := dtoMetricValue(mf.GetType(), m)
			if !ok {
				continue
			}
			f.Samples = append(f.Samples, Sample{
				Labels: convertDTOLabels(m.GetLabel()),
				Value:  v,
			})
		}
	}

	return f, nil
}

func convertDTOLabels(pairs []*dto.LabelPair) []Label {
	if len(pairs) == 0 {
		return nil
	}
	out := make([]Label, 0, len(pairs))
	for _, lp := range pairs {
		out = append(out, Label{Name: lp.GetName(), Value: lp.GetValue()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func dtoMetricValue(t dto.MetricType, m *dto.Metric) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch t {
	case dto.MetricType_COUNTER:
		if m.Counter == nil {
			return 0, false
		}
		return m.Counter.GetValue(), true
	case dto.MetricType_GAUGE:
		if m.Gauge == nil {
			return 0, false
		}
		return m.Gauge.GetValue(), true
	case dto.MetricType_UNTYPED:
		if m.Untyped == nil {
			return 0, false
		}
		return m.Untyped.GetValue(), true
	default:
		return 0, false
	}
}

