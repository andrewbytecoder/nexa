package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FilefdCollector struct{}

func NewFilefdCollector() *FilefdCollector { return &FilefdCollector{} }

func (c *FilefdCollector) Name() string     { return "filefd" }
func (c *FilefdCollector) Describe() string { return "Exposes file descriptor statistics from /proc/sys/fs/file-nr" }

func (c *FilefdCollector) Collect(ctx context.Context) ([]MetricFamily, error) {
	_ = ctx
	f, err := os.Open("/proc/sys/fs/file-nr")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return nil, fmt.Errorf("empty /proc/sys/fs/file-nr")
	}
	parts := strings.Fields(sc.Text())
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected /proc/sys/fs/file-nr format: %q", sc.Text())
	}
	allocated, _ := strconv.ParseFloat(parts[0], 64)
	unused, _ := strconv.ParseFloat(parts[1], 64)
	max, _ := strconv.ParseFloat(parts[2], 64)
	used := allocated - unused

	return []MetricFamily{
		{Name: "node_filefd_allocated", Help: "File descriptor statistics: allocated.", Type: MetricTypeGauge, Samples: []Sample{{Value: allocated}}},
		{Name: "node_filefd_maximum", Help: "File descriptor statistics: maximum.", Type: MetricTypeGauge, Samples: []Sample{{Value: max}}},
		{Name: "node_filefd_used", Help: "File descriptor statistics: used.", Type: MetricTypeGauge, Samples: []Sample{{Value: used}}},
	}, nil
}

