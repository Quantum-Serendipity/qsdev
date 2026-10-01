package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestGeneratedGuardRecognized pins that the package-guard hook the generator
// emits, with and without the sandbox prefix, is the shape posture credits as
// running the guard (claudesettings.RunsScript), for every shell tool.
func TestGeneratedGuardRecognized(t *testing.T) {
	t.Parallel()
	for _, sandbox := range []bool{false, true} {
		gf := mustGenerateSettings(t, allHooksAnswers(sandbox), ecosystem.NewRegistry())
		s, err := claudesettings.Parse(gf.Content)
		if err != nil {
			t.Fatalf("sandbox=%v: parse generated settings: %v", sandbox, err)
		}
		for _, tool := range []string{"Bash", "PowerShell", "Monitor"} {
			if !s.RunsScript(claudesettings.EventPreToolUse, tool, ".claude/hooks/package-guard.py", branding.Get().AppName, claudesettings.GuardHookTimeout) {
				t.Errorf("sandbox=%v: generated guard not recognized as running for %s: %+v",
					sandbox, tool, s.Hooks[claudesettings.EventPreToolUse])
			}
		}
	}
}
