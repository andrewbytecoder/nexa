package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/nexa/pkg/node/collector"
)

// WritePrometheusText serializes MetricFamilies into Prometheus text exposition format.
// This is intended for tests/golden comparisons, not for primary CLI output.
func WritePrometheusText(w io.Writer, families []collector.MetricFamily) error {
	sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })

	for _, f := range families {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n", f.Name, f.Help); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "# TYPE %s %s\n", f.Name, f.Type); err != nil {
			return err
		}

		switch f.Type {
		case collector.MetricTypeHistogram:
			for _, h := range f.Histograms {
				lbl := collector.FormatLabels(h.Labels)
				if _, err := fmt.Fprintf(w, "%s_sum{%s} %f\n", f.Name, lbl, h.Sum); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s_count{%s} %d\n", f.Name, lbl, h.Count); err != nil {
					return err
				}
				for _, b := range h.Buckets {
					l2 := append([]collector.Label(nil), h.Labels...)
					l2 = append(l2, collector.Label{Name: "le", Value: fmt.Sprintf("%g", b.UpperBound)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if _, err := fmt.Fprintf(w, "%s_bucket{%s} %d\n", f.Name, collector.FormatLabels(l2), b.Count); err != nil {
						return err
					}
				}
			}
		case collector.MetricTypeSummary:
			for _, s := range f.Summaries {
				lbl := collector.FormatLabels(s.Labels)
				if _, err := fmt.Fprintf(w, "%s_sum{%s} %f\n", f.Name, lbl, s.Sum); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s_count{%s} %d\n", f.Name, lbl, s.Count); err != nil {
					return err
				}
				for _, q := range s.Quantiles {
					l2 := append([]collector.Label(nil), s.Labels...)
					l2 = append(l2, collector.Label{Name: "quantile", Value: fmt.Sprintf("%g", q.Quantile)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if _, err := fmt.Fprintf(w, "%s{%s} %f\n", f.Name, collector.FormatLabels(l2), q.Value); err != nil {
						return err
					}
				}
			}
		default:
			for _, s := range f.Samples {
				lbl := collector.FormatLabels(s.Labels)
				if lbl != "" {
					if _, err := fmt.Fprintf(w, "%s{%s} %f\n", f.Name, lbl, s.Value); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(w, "%s %f\n", f.Name, s.Value); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

