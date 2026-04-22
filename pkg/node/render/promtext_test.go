package render

import (
	"bytes"
	"testing"

	"github.com/nexa/pkg/node/collector"
)

func TestWritePrometheusText_SimpleGauge(t *testing.T) {
	f := collector.MetricFamily{
		Name: "node_test_metric",
		Help: "Test metric.",
		Type: collector.MetricTypeGauge,
		Samples: []collector.Sample{
			{Labels: []collector.Label{{Name: "a", Value: "b"}}, Value: 1},
		},
	}

	var buf bytes.Buffer
	if err := WritePrometheusText(&buf, []collector.MetricFamily{f}); err != nil {
		t.Fatalf("WritePrometheusText error: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("# HELP node_test_metric")) {
		t.Fatalf("missing HELP line: %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("# TYPE node_test_metric gauge")) {
		t.Fatalf("missing TYPE line: %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`node_test_metric{a="b"} 1.000000`)) {
		t.Fatalf("missing sample line: %q", out)
	}
}

