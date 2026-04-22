package collector

import (
	"context"

	"github.com/shirou/gopsutil/v4/host"
)

type UnameCollector struct{}

func NewUnameCollector() *UnameCollector { return &UnameCollector{} }

func (c *UnameCollector) Name() string     { return "uname" }
func (c *UnameCollector) Describe() string { return "Exposes system information as provided by uname" }

func (c *UnameCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	hi, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return []MetricFamily{
		{
			Name: "node_uname_info",
			Help: "Labeled system information as provided by the uname system call.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{
					Labels: []Label{
						// gopsutil doesn't expose full uname(2) fields; approximate with HostInfoStat.
						{Name: "sysname", Value: hi.OS},
						{Name: "release", Value: hi.KernelVersion},
						{Name: "version", Value: hi.KernelVersion},
						{Name: "machine", Value: hi.KernelArch},
						{Name: "nodename", Value: hi.Hostname},
					},
					Value: 1,
				},
			},
		},
	}, nil
}

