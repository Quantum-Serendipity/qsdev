package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ExitNotJoined is the process exit code for "project not initialized, or
// needs join" (XD-WS5 exit-code table).
const ExitNotJoined = 3

// notJoinedError refuses a mutating command in an un-joined clone. It carries
// ExitNotJoined (gdev's ExitCodeErr contract) and wraps state.ErrNotJoined.
type notJoinedError struct{}

func (*notJoinedError) Error() string {
	b := branding.Get()
	return fmt.Sprintf("this checkout has a committed %s but no local state; run `%s init --yes` to join", b.ConfigFile, b.AppName)
}

// ExitCode satisfies gdev's ExitCodeErr interface.
func (*notJoinedError) ExitCode() int { return ExitNotJoined }

func (*notJoinedError) Unwrap() error { return state.ErrNotJoined }

// RequireJoined refuses to proceed in a fresh clone (committed config, no
// local init state), where regenerating from empty answers would rewrite or
// delete the team's committed files. A never-initialized directory passes.
func RequireJoined(projectRoot string) error {
	needsJoin, err := state.NeedsJoin(projectRoot)
	if err != nil {
		return fmt.Errorf("checking project join state: %w", err)
	}
	if needsJoin {
		return &notJoinedError{}
	}
	return nil
}

// JoinedProject is Project for a command that writes the project's
// generated files: it refuses an un-joined clone (RequireJoined).
func JoinedProject(cmd *cobra.Command) (projectctx.Context, error) {
	pc, err := Project(cmd)
	if err != nil {
		return projectctx.Context{}, err
	}
	if err := RequireJoined(pc.Root); err != nil {
		return projectctx.Context{}, err
	}
	return pc, nil
}
