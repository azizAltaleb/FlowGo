package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestOutboxCollectorExposesCanonicalAndLegacyMirrors(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	collector := NewUnregisteredOutboxCollector(func() OutboxSnapshot {
		return OutboxSnapshot{
			Pending:        3,
			PublishSuccess: 7,
			PublishFailure: 2,
			PublishLagSec:  11,
		}
	})
	registry.MustRegister(collector)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	values := make(map[string]float64, len(families))
	for _, family := range families {
		values[family.GetName()] = metricValue(family)
	}

	pairs := [][2]string{
		{"artificialflow_outbox_pending", "flowgo_outbox_pending"},
		{"artificialflow_outbox_publish_success_total", "flowgo_outbox_publish_success_total"},
		{"artificialflow_outbox_publish_failure_total", "flowgo_outbox_publish_failure_total"},
		{"artificialflow_outbox_publish_lag_seconds", "flowgo_outbox_publish_lag_seconds"},
	}
	for _, pair := range pairs {
		canonical, canonicalOK := values[pair[0]]
		legacy, legacyOK := values[pair[1]]
		if !canonicalOK || !legacyOK {
			t.Fatalf("missing metric pair %q/%q in %#v", pair[0], pair[1], values)
		}
		if canonical != legacy {
			t.Fatalf("metric mirrors differ for %q/%q: %v != %v", pair[0], pair[1], canonical, legacy)
		}
	}
}

func metricValue(family *dto.MetricFamily) float64 {
	if len(family.Metric) == 0 {
		return 0
	}
	metric := family.Metric[0]
	if metric.Gauge != nil {
		return metric.Gauge.GetValue()
	}
	return metric.Counter.GetValue()
}
