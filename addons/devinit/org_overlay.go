package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
)

// recordOrgOverlay records, after a human's init at their own terminal
// (cmdutil.HumanAtTerminal), the org overlay this run resolved as the one
// later runs no human started may read (catalog.RecordOrgConfigPin). An
// agent's init records nothing, so it cannot approve an overlay of its own;
// an overlay below the project or the temporary directory is not recorded
// either, with a warning. Nothing is recorded when init set up no project.
func recordOrgOverlay(cmd *cobra.Command) {
	if !cmdutil.HumanAtTerminal(cmd.InOrStdin()) {
		return
	}
	root, err := cmdutil.ProjectRoot()
	if err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(state.InitStateFile()))); err != nil {
		return
	}
	if _, err := catalog.RecordOrgConfigPin(root); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v; runs an agent starts keep the overlay recorded before, if any\n", err)
	}
}
