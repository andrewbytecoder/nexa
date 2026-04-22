package collector

import (
	"context"
	"time"
)

type TimeCollector struct{}

func NewTimeCollector() *TimeCollector { return &TimeCollector{} }

func (c *TimeCollector) Name() string     { return "time" }
func (c *TimeCollector) Describe() string { return "Exposes the current system time" }

func (c *TimeCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	_ = ctx
	now := time.Now()
	_, offset := now.Zone()

	return []MetricFamily{
		{
			Name: "node_time_seconds",
			Help: "System time in seconds since epoch (1970).",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(now.Unix())},
			},
		},
		{
			Name: "node_time_zone_offset_seconds",
			Help: "System timezone offset in seconds.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(offset)},
			},
		},
	}, nil
}

