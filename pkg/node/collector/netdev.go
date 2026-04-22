package collector

import (
	"context"

	net "github.com/shirou/gopsutil/v4/net"
)

type NetdevCollector struct{}

func NewNetdevCollector() *NetdevCollector { return &NetdevCollector{} }

func (c *NetdevCollector) Name() string     { return "netdev" }
func (c *NetdevCollector) Describe() string { return "Exposes network interface statistics" }

func (c *NetdevCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	stats, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return nil, err
	}
	recvBytes := MetricFamily{Name: "node_network_receive_bytes_total", Help: "Network device statistic receive_bytes.", Type: MetricTypeCounter}
	xmitBytes := MetricFamily{Name: "node_network_transmit_bytes_total", Help: "Network device statistic transmit_bytes.", Type: MetricTypeCounter}
	recvPkts := MetricFamily{Name: "node_network_receive_packets_total", Help: "Network device statistic receive_packets.", Type: MetricTypeCounter}
	xmitPkts := MetricFamily{Name: "node_network_transmit_packets_total", Help: "Network device statistic transmit_packets.", Type: MetricTypeCounter}

	for _, s := range stats {
		labels := []Label{{Name: "device", Value: s.Name}}
		recvBytes.Samples = append(recvBytes.Samples, Sample{Labels: labels, Value: float64(s.BytesRecv)})
		xmitBytes.Samples = append(xmitBytes.Samples, Sample{Labels: labels, Value: float64(s.BytesSent)})
		recvPkts.Samples = append(recvPkts.Samples, Sample{Labels: labels, Value: float64(s.PacketsRecv)})
		xmitPkts.Samples = append(xmitPkts.Samples, Sample{Labels: labels, Value: float64(s.PacketsSent)})
	}

	return []MetricFamily{recvBytes, xmitBytes, recvPkts, xmitPkts}, nil
}

