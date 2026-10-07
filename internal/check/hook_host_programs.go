package check

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/shebang"
)

// HookHostPrograms returns, sorted and each once, the bare program names the
// Claude Code hooks of the project at projectRoot look up on PATH whenever
// they run (see hookPrograms), together with the interpreter that env looks
// up for each project hook script a hook starts as a program (see
// programScripts). It reads the effective settings (claudesettings.Read) and
// returns nothing when they disable all hooks. Paths, programs a hook runs
// only conditionally and anything looked up after the hook changes PATH are
// left out. Nothing is run.
func HookHostPrograms(projectRoot string) ([]string, error) {
	eff, err := claudesettings.Read(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("reading Claude Code settings: %w", err)
	}
	if eff.DisableAllHooks {
		return nil, nil
	}
	var programs []string
	for _, matchers := range eff.Hooks {
		for _, m := range matchers {
			if m.Invalid {
				continue
			}
			for _, h := range m.Hooks {
				if h.Invalid || h.Type != claudesettings.HookTypeCommand {
					continue
				}
				programs = append(programs, commandHostPrograms(projectRoot, h.Command)...)
			}
		}
	}
	slices.Sort(programs)
	return slices.Compact(programs), nil
}

// commandHostPrograms returns the bare program names one hook command looks
// up on PATH (see HookHostPrograms). A hook script that is missing or
// unreadable names none; `qsdev check` reports it.
func commandHostPrograms(projectRoot, command string) []string {
	var programs []string
	for _, run := range hookPrograms(projectRoot, command) {
		if isBareName(run.program) {
			programs = append(programs, run.program)
		}
	}
	for _, script := range programScripts(command) {
		line, err := shebang.Read(filepath.Join(projectRoot, filepath.FromSlash(script)))
		if err != nil {
			continue
		}
		if prog, ok := line.EnvProgram(runtime.GOOS); ok && isBareName(prog) {
			programs = append(programs, prog)
		}
	}
	return programs
}
