package ecosystem_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// mockToolchainChecker implements both EcosystemModule and ToolchainChecker.
// It warns with the project root and the version it was given, so tests can
// see both were passed through.
type mockToolchainChecker struct {
	ecosystem.MockModule
}

func (m *mockToolchainChecker) ToolchainWarnings(_ context.Context, projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.Version == "" {
		return nil
	}
	return []string{projectRoot + " needs " + config.Version}
}

func TestRegistry_ToolchainWarnings(t *testing.T) {
	t.Parallel()

	r := ecosystem.NewRegistry()
	for _, m := range []ecosystem.EcosystemModule{
		&mockToolchainChecker{ecosystem.MockModule{NameVal: "zeta", DisplayNameVal: "Zeta"}},
		&mockToolchainChecker{ecosystem.MockModule{NameVal: "alpha", DisplayNameVal: "Alpha"}},
		&ecosystem.MockModule{NameVal: "plain", DisplayNameVal: "Plain"},
	} {
		if err := r.Register(m); err != nil {
			t.Fatalf("Register(%q): %v", m.Name(), err)
		}
	}
	detected := func(version string) ecosystem.DetectionResult {
		return ecosystem.DetectionResult{Detected: true, SuggestedConfig: ecosystem.ModuleConfig{Version: version}}
	}

	tests := []struct {
		name    string
		results map[string]ecosystem.DetectionResult
		want    []string
	}{
		{name: "no detections", results: nil, want: nil},
		{
			name:    "sorted by module name and prefixed",
			results: map[string]ecosystem.DetectionResult{"zeta": detected("2"), "alpha": detected("1")},
			want:    []string{"Alpha: /p needs 1", "Zeta: /p needs 2"},
		},
		{
			name: "undetected module skipped",
			results: map[string]ecosystem.DetectionResult{
				"alpha": {Detected: false, SuggestedConfig: ecosystem.ModuleConfig{Version: "1"}},
			},
			want: nil,
		},
		{
			name:    "module without ToolchainChecker and unregistered name",
			results: map[string]ecosystem.DetectionResult{"plain": detected("1"), "missing": detected("1")},
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.ToolchainWarnings(context.Background(), "/p", tt.results)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToolchainWarnings() = %q, want %q", got, tt.want)
			}
		})
	}
}
