package devinit

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// exitNotJoined is the process exit code for "project not initialized, or
// needs join" (XD-WS5 exit-code table).
const exitNotJoined = 3

// notJoinedError refuses a mutating command in an un-joined clone. It carries
// exit code 3 (gdev's ExitCodeErr contract) and wraps state.ErrNotJoined.
type notJoinedError struct{}

func (*notJoinedError) Error() string {
	b := branding.Get()
	return fmt.Sprintf("this checkout has a committed %s but no local state; run `%s init --yes` to join", b.ConfigFile, b.AppName)
}

// ExitCode satisfies gdev's ExitCodeErr interface.
func (*notJoinedError) ExitCode() int { return exitNotJoined }

func (*notJoinedError) Unwrap() error { return state.ErrNotJoined }

// requireJoined refuses to proceed in a fresh clone (committed config, no
// local init state), where regenerating from empty answers would rewrite or
// delete the team's committed files. A never-initialized directory passes.
func requireJoined(projectRoot string) error {
	needsJoin, err := state.NeedsJoin(projectRoot)
	if err != nil {
		return fmt.Errorf("checking project join state: %w", err)
	}
	if needsJoin {
		return &notJoinedError{}
	}
	return nil
}
