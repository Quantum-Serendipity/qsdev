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
			return
		}
		// MarshalText rejects the Unknown zero value, so a report or check
		// whose status was never set fails here instead of reading as healthy.
		if _, err := report.OverallStatus.MarshalText(); err != nil {
			t.Errorf("CollectHealth() returned invalid OverallStatus: %v", err)
		}
		for _, c := range report.Checks {
			if _, err := c.Status.MarshalText(); err != nil {
				t.Errorf("CollectHealth() check %q has invalid Status: %v", c.Name, err)
			}
		}
	})

	t.Run("ContentRetentionValid", func(t *testing.T) {
		if _, err := provider.ContentRetention().MarshalText(); err != nil {
			t.Errorf("ContentRetention() returned invalid tier: %v", err)
		}
	})
}
