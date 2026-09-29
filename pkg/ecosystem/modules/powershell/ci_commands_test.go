package powershell_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCICommands_ProvisionsAndEnforcesAnalyzer checks the PSScriptAnalyzer
// scan has a pinned install step before it (F436: the analyzer was never
// provisioned) and fails on findings rather than only printing them.
func TestCICommands_ProvisionsAndEnforcesAnalyzer(t *testing.T) {
	t.Parallel()

	byName := map[string]ecosystem.CICommand{}
	for _, c := range newModule().CICommands(ecosystem.ModuleConfig{}) {
		byName[c.Name] = c
	}
	install, scan := byName["psscriptanalyzer-install"], byName["psscriptanalyzer"]

	tests := []struct {
		name  string
		cmd   ecosystem.CICommand
		phase ecosystem.CIPhase
		want  []string
	}{
		{
			name:  "install",
			cmd:   install,
			phase: ecosystem.CIPhaseInstall,
			want:  []string{`$ErrorActionPreference = "Stop"`, `Install-PSResource -Name PSScriptAnalyzer -Version "[1.25.0]" `, "-Repository PSGallery"},
		},
		{
			name:  "scan",
			cmd:   scan,
			phase: ecosystem.CIPhaseScan,
			want:  []string{`$ErrorActionPreference = "Stop"`, "Import-Module PSScriptAnalyzer -RequiredVersion 1.25.0", "-Severity Error, ParseError", "exit 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.cmd.Phase != tt.phase {
				t.Errorf("phase = %v, want %v", tt.cmd.Phase, tt.phase)
			}
			for _, w := range tt.want {
				if !strings.Contains(tt.cmd.Command, w) {
					t.Errorf("command %q does not contain %q", tt.cmd.Command, w)
				}
			}
			// The script must reach pwsh with its `$` variables intact: the
			// command's only single quotes are the ones around the script.
			res := shelltest.Run(t, t.TempDir(), tt.cmd.Command, map[string]shelltest.Stub{"pwsh": {}})
			wantCall := strings.ReplaceAll(tt.cmd.Command, "'", "")
			if got := strings.Join(res.Calls, "\n"); got != wantCall {
				t.Errorf("pwsh call = %q, want %q", got, wantCall)
			}
		})
	}
}
