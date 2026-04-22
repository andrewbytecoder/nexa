package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUCollector struct{}

func NewCPUCollector() *CPUCollector { return &CPUCollector{} }

func (c *CPUCollector) Name() string     { return "cpu" }
func (c *CPUCollector) Describe() string { return "Exposes CPU statistics" }

func (c *CPUCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	info, _ := cpu.InfoWithContext(ctx)
	times, err := cpu.TimesWithContext(ctx, true)
	if err != nil {
		return nil, err
	}

	cpuSeconds := MetricFamily{
		Name: "node_cpu_seconds_total",
		Help: "Seconds the CPUs spent in each mode.",
		Type: MetricTypeCounter,
	}
	for _, t := range times {
		// node_exporter uses cpu label like "0", "1" and includes mode label.
		cpuLabel := strings.TrimPrefix(t.CPU, "cpu")
		add := func(mode string, v float64) {
			cpuSeconds.Samples = append(cpuSeconds.Samples, Sample{
				Labels: []Label{
					{Name: "cpu", Value: cpuLabel},
					{Name: "mode", Value: mode},
				},
				Value: v,
			})
		}
		add("user", t.User)
		add("nice", t.Nice)
		add("system", t.System)
		add("idle", t.Idle)
		add("iowait", t.Iowait)
		add("irq", t.Irq)
		add("softirq", t.Softirq)
		add("steal", t.Steal)
		add("guest", t.Guest)
		add("guest_nice", t.GuestNice)
	}

	cpuInfo := MetricFamily{
		Name: "node_cpu_info",
		Help: "CPU information.",
		Type: MetricTypeGauge,
	}
	for _, ci := range info {
		model := ci.ModelName
		if len(model) > 64 {
			model = model[:61] + "..."
		}
		flags := strings.Join(ci.Flags, ",")
		cpuInfo.Samples = append(cpuInfo.Samples, Sample{
			Labels: []Label{
				{Name: "cpu", Value: fmt.Sprintf("%d", ci.CPU)},
				{Name: "vendor_id", Value: ci.VendorID},
				{Name: "family", Value: ci.Family},
				{Name: "model", Value: ci.Model},
				{Name: "model_name", Value: model},
				{Name: "stepping", Value: fmt.Sprintf("%d", ci.Stepping)},
				{Name: "microcode", Value: ci.Microcode},
				{Name: "mhz", Value: fmt.Sprintf("%.0f", ci.Mhz)},
				{Name: "cache_size", Value: fmt.Sprintf("%d", ci.CacheSize)},
				{Name: "flags", Value: flags},
			},
			Value: 1,
		})
	}

	return []MetricFamily{
		cpuSeconds,
		cpuInfo,
	}, nil
}

