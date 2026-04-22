package collector

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestDTOToNexa_Gauge(t *testing.T) {
	mf := &dto.MetricFamily{
		Name: protoString("node_test_gauge"),
		Help: protoString("Test gauge."),
		Type: dto.MetricType_GAUGE.Enum(),
		Metric: []*dto.Metric{
			{
				Label: []*dto.LabelPair{{Name: protoString("a"), Value: protoString("b")}},
				Gauge: &dto.Gauge{Value: protoFloat64(2)},
			},
		},
	}

	out, err := DTOToNexa([]*dto.MetricFamily{mf})
	if err != nil {
		t.Fatalf("DTOToNexa error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 family, got %d", len(out))
	}
	if out[0].Name != "node_test_gauge" || out[0].Type != MetricTypeGauge {
		t.Fatalf("unexpected family: %+v", out[0])
	}
	if len(out[0].Samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(out[0].Samples))
	}
	if out[0].Samples[0].Value != 2 {
		t.Fatalf("unexpected value: %v", out[0].Samples[0].Value)
	}
}

func protoString(s string) *string { return &s }
func protoFloat64(v float64) *float64 { return &v }

