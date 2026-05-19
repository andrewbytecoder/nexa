package render

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/nexa/pkg/node/collector"
	"github.com/olekukonko/tablewriter"
)

type Options struct {
	ShowSamples bool
	Limit       int
	Humanize    bool
	MetricRegex string
	LabelEquals []string // key=value, repeatable
	Output      string   // plain|table|prom
}

func PrintMetricFamilies(w io.Writer, families []collector.MetricFamily, opt Options) error {
	if opt.Limit <= 0 {
		opt.Limit = 2000
	}
	if opt.Output == "" {
		opt.Output = "plain"
	}
	filtered := make([]collector.MetricFamily, 0, len(families))
	for _, f := range families {
		// TODO: apply regex and label filters (in next todo).
		filtered = append(filtered, f)
	}

	switch opt.Output {
	case "plain":
		if opt.ShowSamples {
			return printSamplesPlain(w, filtered, opt.Limit, opt.Humanize)
		}
		return printFamiliesSummaryPlain(w, filtered, opt.Humanize)
	case "table":
		if opt.ShowSamples {
			return printSamplesTable(w, filtered, opt.Limit, opt.Humanize)
		}
		return printFamiliesSummaryTable(w, filtered, opt.Humanize)
	case "prom":
		return WritePrometheusText(w, filtered)
	default:
		return fmt.Errorf("unknown output format: %q (want: plain, table, prom)", opt.Output)
	}
}

func printFamiliesSummaryTable(w io.Writer, families []collector.MetricFamily, humanize bool) error {
	t := tablewriter.NewWriter(w)
	t.Header([]string{"Metric", "Type", "Help", "Samples", "Example"})

	for _, f := range families {
		lbl, val := f.Example()

		// Override numeric examples with human-readable formatting where possible.
		if humanize && (f.Type == collector.MetricTypeGauge || f.Type == collector.MetricTypeCounter || f.Type == collector.MetricTypeUntyped) && len(f.Samples) > 0 {
			val = formatValueHuman(f.Name, f.Samples[0].Value)
		}

		ex := val
		if lbl != "" {
			ex = fmt.Sprintf("{%s} %s", lbl, val)
		}
		_ = t.Append([]string{
			f.Name,
			string(f.Type),
			f.Help,
			fmt.Sprintf("%d", f.SamplesCount()),
			ex,
		})
	}
	return t.Render()
}

func printFamiliesSummaryPlain(w io.Writer, families []collector.MetricFamily, humanize bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "METRIC\tTYPE\tSAMPLES\tEXAMPLE\tHELP")

	for _, f := range families {
		lbl, val := f.Example()

		if humanize && (f.Type == collector.MetricTypeGauge || f.Type == collector.MetricTypeCounter || f.Type == collector.MetricTypeUntyped) && len(f.Samples) > 0 {
			val = formatValueHuman(f.Name, f.Samples[0].Value)
		}

		ex := val
		if lbl != "" {
			ex = fmt.Sprintf("{%s} %s", lbl, val)
		}

		help := oneLine(strings.TrimSpace(f.Help))
		if len(help) > 80 {
			help = help[:77] + "..."
		}

		_, _ = fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n", f.Name, string(f.Type), f.SamplesCount(), ex, help)
	}
	return tw.Flush()
}

func printSamplesTable(w io.Writer, families []collector.MetricFamily, limit int, humanize bool) error {
	t := tablewriter.NewWriter(w)
	t.Header([]string{"Metric", "Labels", "Value"})

	printed := 0
	truncated := false

	emit := func(metric string, labels string, value string) {
		if printed >= limit {
			truncated = true
			return
		}
		_ = t.Append([]string{metric, labels, value})
		printed++
	}

	for _, f := range families {
		switch f.Type {
		case collector.MetricTypeHistogram:
			for _, h := range f.Histograms {
				lbl := collector.FormatLabels(h.Labels)
				sumMetric := f.Name + "_sum"
				countMetric := f.Name + "_count"
				if humanize {
					emit(sumMetric, lbl, formatValueHuman(sumMetric, h.Sum))
					emit(countMetric, lbl, humanizeNumber(float64(h.Count)))
				} else {
					emit(sumMetric, lbl, fmt.Sprintf("%f", h.Sum))
					emit(countMetric, lbl, fmt.Sprintf("%d", h.Count))
				}
				for _, b := range h.Buckets {
					l2 := append([]collector.Label(nil), h.Labels...)
					l2 = append(l2, collector.Label{Name: "le", Value: fmt.Sprintf("%g", b.UpperBound)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if humanize {
						emit(f.Name+"_bucket", collector.FormatLabels(l2), humanizeNumber(float64(b.Count)))
					} else {
						emit(f.Name+"_bucket", collector.FormatLabels(l2), fmt.Sprintf("%d", b.Count))
					}
				}
			}
		case collector.MetricTypeSummary:
			for _, s := range f.Summaries {
				lbl := collector.FormatLabels(s.Labels)
				sumMetric := f.Name + "_sum"
				countMetric := f.Name + "_count"
				if humanize {
					emit(sumMetric, lbl, formatValueHuman(sumMetric, s.Sum))
					emit(countMetric, lbl, humanizeNumber(float64(s.Count)))
				} else {
					emit(sumMetric, lbl, fmt.Sprintf("%f", s.Sum))
					emit(countMetric, lbl, fmt.Sprintf("%d", s.Count))
				}
				for _, q := range s.Quantiles {
					l2 := append([]collector.Label(nil), s.Labels...)
					l2 = append(l2, collector.Label{Name: "quantile", Value: fmt.Sprintf("%g", q.Quantile)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if humanize {
						emit(f.Name, collector.FormatLabels(l2), formatValueHuman(f.Name, q.Value))
					} else {
						emit(f.Name, collector.FormatLabels(l2), fmt.Sprintf("%f", q.Value))
					}
				}
			}
		default:
			for _, s := range f.Samples {
				if humanize {
					emit(f.Name, collector.FormatLabels(s.Labels), formatValueHuman(f.Name, s.Value))
				} else {
					emit(f.Name, collector.FormatLabels(s.Labels), fmt.Sprintf("%f", s.Value))
				}
			}
		}
		if truncated {
			break
		}
	}

	if err := t.Render(); err != nil {
		return err
	}
	if truncated {
		fmt.Fprintf(w, "\n(truncated to %d rows; use --limit or filters)\n", limit)
	}
	return nil
}

func printSamplesPlain(w io.Writer, families []collector.MetricFamily, limit int, humanize bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "METRIC\tLABELS\tVALUE")

	printed := 0
	truncated := false

	emit := func(metric string, labels string, value string) {
		if printed >= limit {
			truncated = true
			return
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", metric, labels, value)
		printed++
	}

	for _, f := range families {
		switch f.Type {
		case collector.MetricTypeHistogram:
			for _, h := range f.Histograms {
				lbl := collector.FormatLabels(h.Labels)
				sumMetric := f.Name + "_sum"
				countMetric := f.Name + "_count"
				if humanize {
					emit(sumMetric, lbl, formatValueHuman(sumMetric, h.Sum))
					emit(countMetric, lbl, humanizeNumber(float64(h.Count)))
				} else {
					emit(sumMetric, lbl, fmt.Sprintf("%f", h.Sum))
					emit(countMetric, lbl, fmt.Sprintf("%d", h.Count))
				}
				for _, b := range h.Buckets {
					l2 := append([]collector.Label(nil), h.Labels...)
					l2 = append(l2, collector.Label{Name: "le", Value: fmt.Sprintf("%g", b.UpperBound)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if humanize {
						emit(f.Name+"_bucket", collector.FormatLabels(l2), humanizeNumber(float64(b.Count)))
					} else {
						emit(f.Name+"_bucket", collector.FormatLabels(l2), fmt.Sprintf("%d", b.Count))
					}
				}
			}
		case collector.MetricTypeSummary:
			for _, s := range f.Summaries {
				lbl := collector.FormatLabels(s.Labels)
				sumMetric := f.Name + "_sum"
				countMetric := f.Name + "_count"
				if humanize {
					emit(sumMetric, lbl, formatValueHuman(sumMetric, s.Sum))
					emit(countMetric, lbl, humanizeNumber(float64(s.Count)))
				} else {
					emit(sumMetric, lbl, fmt.Sprintf("%f", s.Sum))
					emit(countMetric, lbl, fmt.Sprintf("%d", s.Count))
				}
				for _, q := range s.Quantiles {
					l2 := append([]collector.Label(nil), s.Labels...)
					l2 = append(l2, collector.Label{Name: "quantile", Value: fmt.Sprintf("%g", q.Quantile)})
					sort.Slice(l2, func(i, j int) bool { return l2[i].Name < l2[j].Name })
					if humanize {
						emit(f.Name, collector.FormatLabels(l2), formatValueHuman(f.Name, q.Value))
					} else {
						emit(f.Name, collector.FormatLabels(l2), fmt.Sprintf("%f", q.Value))
					}
				}
			}
		default:
			for _, s := range f.Samples {
				if humanize {
					emit(f.Name, collector.FormatLabels(s.Labels), formatValueHuman(f.Name, s.Value))
				} else {
					emit(f.Name, collector.FormatLabels(s.Labels), fmt.Sprintf("%f", s.Value))
				}
			}
		}
		if truncated {
			break
		}
	}

	if err := tw.Flush(); err != nil {
		return err
	}
	if truncated {
		fmt.Fprintf(w, "\n(truncated to %d rows; use --limit or filters)\n", limit)
	}
	return nil
}

func PrintCollectorList(w io.Writer, rows []collector.CollectorStatus) error {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	t := tablewriter.NewWriter(w)
	t.Header([]string{"Collector", "Implemented", "Description"})

	for _, r := range rows {
		impl := "no"
		if r.Implemented {
			impl = "yes"
		}
		_ = t.Append([]string{r.Name, impl, r.Description})
	}
	return t.Render()
}

func PrintCollectorListPlain(w io.Writer, rows []collector.CollectorStatus) error {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "COLLECTOR\tIMPL\tDESCRIPTION")
	for _, r := range rows {
		impl := "no"
		if r.Implemented {
			impl = "yes"
		}
		desc := oneLine(strings.TrimSpace(r.Description))
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Name, impl, desc)
	}
	return tw.Flush()
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

