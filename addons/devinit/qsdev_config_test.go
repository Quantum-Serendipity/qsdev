package devinit

import (
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Regression: init must record qsdev_version as a lower-bound constraint,
// never the running binary's exact version, so a teammate or CI runner on a
// newer patch release still passes the binary_compat check.
func TestBuildQsdevConfig_QsdevVersionIsLowerBound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		binaryVersion string
		want          string
	}{
		{"release with build metadata", "0.8.0+abc1234", ">= 0.8.0"},
		{"dev build writes no constraint", "dev", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := qsdevconfig.AnswersToConfig(types.WizardAnswers{}, tt.binaryVersion)
			if cfg.QsdevVersion != tt.want {
				t.Fatalf("QsdevVersion = %q, want %q", cfg.QsdevVersion, tt.want)
			}
			if err := qsdevconfig.CheckBinaryVersion(cfg.QsdevVersion, "0.8.1+aaa"); err != nil {
				t.Errorf("newer patch release rejected: %v", err)
			}
		})
	}
}
