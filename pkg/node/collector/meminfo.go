package collector

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"
)

type MeminfoCollector struct{}

func NewMeminfoCollector() *MeminfoCollector { return &MeminfoCollector{} }

func (c *MeminfoCollector) Name() string     { return "meminfo" }
func (c *MeminfoCollector) Describe() string { return "Exposes memory statistics" }

func (c *MeminfoCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Align with node_exporter meminfo naming where possible.
	// For full parity we will expand this with /proc/meminfo parsing later.
	mf := []MetricFamily{
		{
			Name: "node_memory_MemTotal_bytes",
			Help: "Memory information field MemTotal_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Total)},
			},
		},
		{
			Name: "node_memory_MemFree_bytes",
			Help: "Memory information field MemFree_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Free)},
			},
		},
		{
			Name: "node_memory_Active_bytes",
			Help: "Memory information field Active_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Active)},
			},
		},
		{
			Name: "node_memory_Inactive_bytes",
			Help: "Memory information field Inactive_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Inactive)},
			},
		},
		{
			Name: "node_memory_MemAvailable_bytes",
			Help: "Memory information field MemAvailable_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Available)},
			},
		},
		{
			Name: "node_memory_Buffers_bytes",
			Help: "Memory information field Buffers_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Buffers)},
			},
		},
		{
			Name: "node_memory_Cached_bytes",
			Help: "Memory information field Cached_bytes.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{Value: float64(v.Cached)},
			},
		},
	}
	return mf, nil
}

