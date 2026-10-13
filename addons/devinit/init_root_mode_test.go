package devinit

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestInitUpdateTargetsCwdNotAncestor is the U02-07 regression: `init
// --update` in an empty, non-git directory below an ancestor holding the
// state directory acts on the working directory (it is marked MarkRootHere)
// and never names the ancestor, whether the ancestor's marker is trusted or
// planted in a world-writable directory.
func TestInitUpdateTargetsCwdNotAncestor(t *testing.T) {
	b := branding.Get()
	tests := []struct {
		name     string
		shared   bool
		unixOnly bool
	}{
		{name: "trusted ancestor"},
		{name: "ancestor in a world-writable parent", shared: true, unixOnly: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("marker trust is ACL-based on Windows and out of scope")
			}
			base := testutil.MarkerFreeTempDir(t)
			anc := filepath.Join(base, "anc")
			plantPaths(t, base, "anc/"+b.StateDir+"/", "anc/child/")
			if tt.shared {
				chmodShared(t, anc)
			}
			child := filepath.Join(anc, "child")

			out, err := executeInitCmd(t, child, "--update", "--dry-run", "--yes")
			if err == nil || !strings.Contains(err.Error(), "no saved answers") {
				t.Fatalf("init --update: err = %v, want no saved answers", err)
			}
			if !strings.Contains(err.Error(), child) {
				t.Errorf("init --update error %q does not name the working directory %s", err, child)
			}
			ancState := filepath.Join(anc, b.StateDir)
			if strings.Contains(err.Error(), ancState) || strings.Contains(out, ancState) {
				t.Errorf("init --update acted on the ancestor %s:\nerr: %v\nout: %s", anc, err, out)
			}
		})
	}
}

// TestInitCommandsAreRootHere pins that devinit's init resolves its project
// with projectctx.Here, and the commands that act on an existing project do
// not.
func TestInitCommandsAreRootHere(t *testing.T) {
	t.Parallel()
	if got := cmdutil.RootMode(initCmd()); got != projectctx.Here {
		t.Errorf("RootMode(init) = %v, want Here", got)
	}
	if got := cmdutil.RootMode(trialCmd()); got != projectctx.Enclosing {
		t.Errorf("RootMode(trial) = %v, want Enclosing", got)
	}
}
