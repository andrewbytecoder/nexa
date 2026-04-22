package collector

import (
	"context"
	"sort"

	"github.com/shirou/gopsutil/v4/disk"
)

type DiskstatsCollector struct{}

func NewDiskstatsCollector() *DiskstatsCollector { return &DiskstatsCollector{} }

func (c *DiskstatsCollector) Name() string     { return "diskstats" }
func (c *DiskstatsCollector) Describe() string { return "Exposes disk I/O statistics" }

func (c *DiskstatsCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	m, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		return nil, err
	}
	devs := make([]string, 0, len(m))
	for k := range m {
		devs = append(devs, k)
	}
	sort.Strings(devs)

	readBytes := MetricFamily{Name: "node_disk_read_bytes_total", Help: "The total number of bytes read successfully.", Type: MetricTypeCounter}
	writtenBytes := MetricFamily{Name: "node_disk_written_bytes_total", Help: "The total number of bytes written successfully.", Type: MetricTypeCounter}
	readsCompleted := MetricFamily{Name: "node_disk_reads_completed_total", Help: "The total number of reads completed successfully.", Type: MetricTypeCounter}
	writesCompleted := MetricFamily{Name: "node_disk_writes_completed_total", Help: "The total number of writes completed successfully.", Type: MetricTypeCounter}

	for _, dev := range devs {
		s := m[dev]
		labels := []Label{{Name: "device", Value: s.Name}}
		readBytes.Samples = append(readBytes.Samples, Sample{Labels: labels, Value: float64(s.ReadBytes)})
		writtenBytes.Samples = append(writtenBytes.Samples, Sample{Labels: labels, Value: float64(s.WriteBytes)})
		readsCompleted.Samples = append(readsCompleted.Samples, Sample{Labels: labels, Value: float64(s.ReadCount)})
		writesCompleted.Samples = append(writesCompleted.Samples, Sample{Labels: labels, Value: float64(s.WriteCount)})
	}

	return []MetricFamily{readBytes, writtenBytes, readsCompleted, writesCompleted}, nil
}

