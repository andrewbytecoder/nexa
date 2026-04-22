package collector

import (
	"context"

	up "github.com/nexa/pkg/node/upstream"
)

type UpstreamCollector struct {
	name string
	desc string
}

func NewUpstreamCollector(name, desc string) *UpstreamCollector {
	return &UpstreamCollector{name: name, desc: desc}
}

func (c *UpstreamCollector) Name() string     { return c.name }
func (c *UpstreamCollector) Describe() string { return c.desc }

func (c *UpstreamCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	dto, err := up.CollectDTO(ctx, []string{c.name})
	if err != nil {
		return nil, err
	}
	return DTOToNexa(dto)
}

