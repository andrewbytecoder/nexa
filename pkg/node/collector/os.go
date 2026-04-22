package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

type OSCollector struct{}

func NewOSCollector() *OSCollector { return &OSCollector{} }

func (c *OSCollector) Name() string     { return "os" }
func (c *OSCollector) Describe() string { return "Expose OS release info" }

func (c *OSCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	hi, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}
	platform := hi.Platform
	if platform == "" {
		platform = "unknown"
	}
	version := hi.PlatformVersion
	if version == "" {
		version = "unknown"
	}
	pretty := strings.TrimSpace(fmt.Sprintf("%s %s", platform, version))
	if len(pretty) > 48 {
		pretty = pretty[:45] + "..."
	}

	return []MetricFamily{
		{
			Name: "node_os_info",
			Help: "Operating system information.",
			Type: MetricTypeGauge,
			Samples: []Sample{
				{
					Labels: []Label{
						{Name: "name", Value: platform},
						{Name: "version", Value: version},
						{Name: "pretty_name", Value: pretty},
					},
					Value: 1,
				},
			},
		},
	}, nil
}

