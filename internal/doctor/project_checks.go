package doctor

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// HookRequiredBy labels a tool doctor requires because the project's Claude
// Code hooks run it. The hooks look it up on the PATH Claude Code starts them
// with, which doctor can only assume is the invoking shell's (W0N-03).
const HookRequiredBy = "needed by Claude Code hooks (resolved on this shell's PATH)"

// ProjectChecks returns the host tool checks for the project at projectRoot:
// DefaultChecks, with every program the project's Claude Code hooks look up
// on PATH (check.HookHostPrograms) required, so a missing hook interpreter,
// or one below its floor, fails. Outside a project ("") it is DefaultChecks
// alone. On an error reading the hooks DefaultChecks is returned with it.
// Every doctor verdict for a project (the command, the MCP tool) uses it.
func ProjectChecks(projectRoot string) ([]ToolCheck, error) {
	checks := DefaultChecks()
	if projectRoot == "" {
		return checks, nil
	}
	programs, err := check.HookHostPrograms(projectRoot)
	if err != nil {
		return checks, fmt.Errorf("finding the programs the Claude Code hooks need: %w", err)
	}
	return RequireBinaries(checks, programs, HookRequiredBy), nil
}

// HookProgramsWarning is the warning every doctor verdict shows when
// ProjectChecks failed with err: the hook programs were not checked, so a
// pass does not cover them.
func HookProgramsWarning(err error) string {
	return fmt.Sprintf("Could not read the Claude Code settings; hook programs were not checked: %v (run '%s check')",
		err, branding.Get().AppName)
}
