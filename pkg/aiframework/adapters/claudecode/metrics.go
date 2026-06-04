package claudecode

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func (a *Adapter) EmitEvent(_ context.Context, _ aiframework.MetricEvent) error {
	return nil
}

func (a *Adapter) CollectHealth(_ context.Context) (*aiframework.HealthReport, error) {
	return &aiframework.HealthReport{
		FrameworkID:   aiframework.ClaudeCode,
		OverallStatus: aiframework.StatusHealthy,
	}, nil
}

func (a *Adapter) ContentRetention() aiframework.ContentTier {
	return aiframework.ContentFull
}
