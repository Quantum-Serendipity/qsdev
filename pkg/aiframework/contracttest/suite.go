package contracttest

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

type ContractAdapters struct {
	Detection aiframework.DetectionAdapter
	Config    aiframework.ConfigRenderer
	Hooks     aiframework.HookDeployer
	Registry  aiframework.RegistryClient
	Metrics   aiframework.MetricsProvider
	State     aiframework.StateBackend
	Tools     aiframework.ToolAdapter
}

type ContractFixtures struct {
	PresentRoot string
	AbsentRoot  string
	PolicyInput *aiframework.PolicyInput
}

func RunAllContractTests(t *testing.T, adapters ContractAdapters, fixtures ContractFixtures) {
	t.Helper()

	if adapters.Detection != nil {
		t.Run("Detection", func(t *testing.T) {
			TestDetectionAdapter(t, adapters.Detection, fixtures)
		})
	}
	if adapters.Config != nil {
		t.Run("Config", func(t *testing.T) {
			TestConfigRenderer(t, adapters.Config, fixtures)
		})
	}
	if adapters.Hooks != nil {
		t.Run("Hooks", func(t *testing.T) {
			TestHookDeployer(t, adapters.Hooks, fixtures)
		})
	}
	if adapters.Registry != nil {
		t.Run("Registry", func(t *testing.T) {
			TestRegistryClient(t, adapters.Registry, fixtures)
		})
	}
	if adapters.Metrics != nil {
		t.Run("Metrics", func(t *testing.T) {
			TestMetricsProvider(t, adapters.Metrics, fixtures)
		})
	}
	if adapters.State != nil {
		t.Run("State", func(t *testing.T) {
			TestStateBackend(t, adapters.State, fixtures)
		})
	}
	if adapters.Tools != nil {
		t.Run("Tools", func(t *testing.T) {
			TestToolAdapter(t, adapters.Tools, fixtures)
		})
	}
}
