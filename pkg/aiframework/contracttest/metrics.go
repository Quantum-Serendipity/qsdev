package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestMetricsProvider(t *testing.T, provider aiframework.MetricsProvider, fixtures ContractFixtures) {
	t.Helper()

	t.Run("EmitEventNoError", func(t *testing.T) {
		event := aiframework.MetricEvent{
			Timestamp:   time.Now(),
			Action:      "test",
			Category:    "test",
			Severity:    "info",
			FrameworkID: provider.FrameworkID(),
		}
		if err := provider.EmitEvent(context.Background(), event); err != nil {
			t.Errorf("EmitEvent() error: %v", err)
		}
	})

	t.Run("CollectHealthValid", func(t *testing.T) {
		report, err := provider.CollectHealth(context.Background())
		if err != nil {
			t.Fatalf("CollectHealth() error: %v", err)
		}
		if report == nil {
			t.Fatal("CollectHealth() returned nil")
		}
		status := report.OverallStatus.String()
		if status == "unknown" {
			t.Error("CollectHealth() returned unknown OverallStatus")
		}
	})

	t.Run("ContentRetentionValid", func(t *testing.T) {
		tier := provider.ContentRetention()
		if tier.String() == "unknown" {
			t.Error("ContentRetention() returned unknown tier")
		}
	})
}
