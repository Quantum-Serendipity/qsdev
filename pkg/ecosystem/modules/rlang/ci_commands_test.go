package rlang_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/rlang"
)

// TestCICommands_GatedOnRenv checks only renv projects get the renv restore
// and status commands (F436: renv::restore ran even without renv).
func TestCICommands_GatedOnRenv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pm   string
		want []string
	}{
		{pm: "renv", want: []string{"renv-restore", "renv-status"}},
		{pm: ""},
	}
	for _, tt := range tests {
		t.Run("pm="+tt.pm, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, c := range (&rlang.Module{}).CICommands(ecosystem.ModuleConfig{PackageManager: tt.pm}) {
				got = append(got, c.Name)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("CI commands = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCICommands_RScriptReachesR runs each renv command with a stub Rscript:
// the R expression must reach R verbatim, so the shell must not expand
// `$synchronized` (which would leave `renv::status()` and a status check
// that always passes).
func TestCICommands_RScriptReachesR(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"renv-restore": "Rscript -e renv::restore()",
		"renv-status":  "Rscript -e if (!isTRUE(renv::status()$synchronized)) quit(status = 1)",
	}
	for _, c := range (&rlang.Module{}).CICommands(ecosystem.ModuleConfig{PackageManager: "renv"}) {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			res := shelltest.Run(t, t.TempDir(), c.Command, map[string]shelltest.Stub{"Rscript": {}})
			if got := strings.Join(res.Calls, "\n"); got != want[c.Name] {
				t.Errorf("Rscript call = %q, want %q", got, want[c.Name])
			}
		})
	}
}
