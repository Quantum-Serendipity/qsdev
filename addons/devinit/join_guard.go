package devinit

import "github.com/Quantum-Serendipity/qsdev/internal/cmdutil"

// exitNotJoined is the process exit code for "project not initialized, or
// needs join" (XD-WS5 exit-code table).
const exitNotJoined = cmdutil.ExitNotJoined

// requireJoined refuses to proceed in a fresh clone; see
// cmdutil.RequireJoined.
func requireJoined(projectRoot string) error {
	return cmdutil.RequireJoined(projectRoot)
}
