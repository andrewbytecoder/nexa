package upstream

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	nexp "github.com/prometheus/node_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var initOnce sync.Once

func ensureUpstreamDefaultsParsed() {
	initOnce.Do(func() {
		// node_exporter collectors register kingpin flags with defaults (enabled/disabled, procfs paths, etc).
		// Without parsing, bool flags default to false and NewNodeCollector may think collectors are disabled.
		_, _ = kingpin.CommandLine.Parse([]string{})
	})
}

// CollectDTO uses upstream node_exporter collectors to gather Prometheus dto metric families.
// filters corresponds to node_exporter collector names; empty means "all enabled" upstream.
func CollectDTO(ctx context.Context, filters []string) ([]*dto.MetricFamily, error) {
	_ = ctx

	ensureUpstreamDefaultsParsed()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
	nc, err := nexp.NewNodeCollector(logger, filters...)
	if err != nil {
		return nil, err
	}

	reg := prometheus.NewRegistry()
	if err := reg.Register(nc); err != nil {
		return nil, err
	}

	return reg.Gather()
}
