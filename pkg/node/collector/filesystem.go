package collector

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
)

type FilesystemCollector struct{}

func NewFilesystemCollector() *FilesystemCollector { return &FilesystemCollector{} }

func (c *FilesystemCollector) Name() string     { return "filesystem" }
func (c *FilesystemCollector) Describe() string { return "Exposes filesystem statistics" }

func (c *FilesystemCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	parts, err := disk.PartitionsWithContext(ctx, true)
	if err != nil {
		return nil, err
	}

	size := MetricFamily{Name: "node_filesystem_size_bytes", Help: "Filesystem size in bytes.", Type: MetricTypeGauge}
	avail := MetricFamily{Name: "node_filesystem_avail_bytes", Help: "Filesystem space available to non-root users in bytes.", Type: MetricTypeGauge}
	free := MetricFamily{Name: "node_filesystem_free_bytes", Help: "Filesystem free space in bytes.", Type: MetricTypeGauge}
	files := MetricFamily{Name: "node_filesystem_files", Help: "Filesystem total number of inodes.", Type: MetricTypeGauge}
	filesFree := MetricFamily{Name: "node_filesystem_files_free", Help: "Filesystem total number of free inodes.", Type: MetricTypeGauge}

	for _, p := range parts {
		u, uerr := disk.UsageWithContext(ctx, p.Mountpoint)
		if uerr != nil {
			continue
		}
		labels := []Label{
			{Name: "device", Value: p.Device},
			{Name: "fstype", Value: p.Fstype},
			{Name: "mountpoint", Value: p.Mountpoint},
		}
		size.Samples = append(size.Samples, Sample{Labels: labels, Value: float64(u.Total)})
		avail.Samples = append(avail.Samples, Sample{Labels: labels, Value: float64(u.Free)}) // best-effort
		free.Samples = append(free.Samples, Sample{Labels: labels, Value: float64(u.Free)})
		files.Samples = append(files.Samples, Sample{Labels: labels, Value: float64(u.InodesTotal)})
		filesFree.Samples = append(filesFree.Samples, Sample{Labels: labels, Value: float64(u.InodesFree)})
	}

	return []MetricFamily{size, avail, free, files, filesFree}, nil
}

