package swift_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCICommands_EnforcePackageResolved checks every SwiftPM CI command uses
// the versions pinned in Package.resolved and fails rather than re-resolving
// (F436: plain `swift package resolve` rewrites a stale Package.resolved).
func TestCICommands_EnforcePackageResolved(t *testing.T) {
	t.Parallel()

	cmds := newModule().CICommands(ecosystem.ModuleConfig{})
	if len(cmds) == 0 {
		t.Fatal("no Swift CI commands")
	}
	for _, c := range cmds {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			if !strings.HasSuffix(c.Command, " --force-resolved-versions") {
				t.Errorf("%s = %q, want --force-resolved-versions", c.Name, c.Command)
			}
		})
	}
}
