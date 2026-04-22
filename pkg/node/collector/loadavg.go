package collector

import (
	"context"

	"github.com/shirou/gopsutil/v4/load"
)

type LoadavgCollector struct{}

func NewLoadavgCollector() *LoadavgCollector { return &LoadavgCollector{} }

func (c *LoadavgCollector) Name() string     { return "loadavg" }
func (c *LoadavgCollector) Describe() string { return "Exposes load average" }

func (c *LoadavgCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return []MetricFamily{
		{
			Name: "node_load1",
			Help: "1m load average.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: avg.Load1},
			},
		},
		{
			Name: "node_load5",
			Help: "5m load average.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: avg.Load5},
			},
		},
		{
			Name: "node_load15",
			Help: "15m load average.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: avg.Load15},
			},
		},
	}, nil
}

