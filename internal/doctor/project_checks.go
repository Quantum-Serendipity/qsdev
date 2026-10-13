package doctor

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// HookRequiredBy labels a tool doctor requires because the project's Claude
// Code hooks run it. The hooks look it up on the PATH Claude Code starts them
// with, which doctor can only assume is the invoking shell's (W0N-03).
const HookRequiredBy = "needed by Claude Code hooks (resolved on this shell's PATH)"

// ProjectChecks returns the host tool checks for the project at projectRoot:
// DefaultChecks, with every program the project's Claude Code hooks look up
// on PATH (check.HookHostPrograms) required, so a missing hook interpreter,
// or one below its floor, fails. When the hooks run the CLI itself, the copy
// they find on PATH must also satisfy the project's qsdev_version
// (withSelfCheck). Outside a project ("") it is DefaultChecks alone. On an
// error reading the hooks DefaultChecks is returned with it; on one reading
// the project config the checks are returned, without the version
// requirement, with it.
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
	checks = RequireBinaries(checks, programs, HookRequiredBy)
	cfg, err := config.CommittedConfig(projectRoot)
	var constraint string
	if cfg != nil {
		constraint = cfg.QsdevVersion
	}
	checks = withSelfCheck(checks, constraint, selfExecutable(), runtime.GOOS, os.Getenv("PATHEXT"))
	if err != nil {
		return checks, fmt.Errorf("reading the %s version the project requires: %w", branding.Get().AppName, err)
	}
	return checks, nil
}

// selfExecutable is the path of the running CLI, or "" when it is unknown.
func selfExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}

// withSelfCheck turns the lookup-only check RequireBinaries gave the CLI's
// own program, when the hooks run it, into one that probes its --version
// (the CLI's own binary, so running it is no more than the hooks do) and
// requires constraint, the project's qsdev_version, so a stale copy first on
// PATH, which the fail-closed selfprotect hook would run, fails doctor. Its
// fix is a PATH hint naming exe, the running binary: no package manager
// provides the CLI. checks is not modified.
func withSelfCheck(checks []ToolCheck, constraint, exe, goos, pathExt string) []ToolCheck {
	app := branding.Get().AppName
	self := programKey(app, goos, pathExt)
	i := slices.IndexFunc(checks, func(c ToolCheck) bool { return programKey(c.Binary, goos, pathExt) == self })
	if i < 0 {
		return checks
	}
	out := slices.Clone(checks)
	out[i].VersionFlag = "--version"
	out[i].ParseVersion = parseSelfVersion
	out[i].Constraint = constraint
	out[i].AutoInstall = nil
	if exe == "" {
		out[i].PathHint = "put the " + app + " you are running first on PATH"
	} else {
		out[i].PathHint = "put the directory of " + exe + " first on PATH"
	}
	return out
}

// parseSelfVersion extracts the version from "<app> version <v>", the first
// line the CLI prints for --version.
func parseSelfVersion(raw string) string {
	fields := strings.Fields(toolcheck.FirstLine(raw))
	if len(fields) == 3 && fields[1] == "version" {
		return fields[2]
	}
	return ""
}

// HookProgramsWarning is the warning every doctor verdict shows when
// ProjectChecks failed with err: the hook programs were not checked, so a
// pass does not cover them.
func HookProgramsWarning(err error) string {
	return fmt.Sprintf("Could not read what the Claude Code hooks need; hook programs were not checked: %v (run '%s check')",
		err, branding.Get().AppName)
}
