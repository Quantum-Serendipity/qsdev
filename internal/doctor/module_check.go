package doctor

import (
	"fmt"
	"io"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

// Module check statuses reported in ModuleCheckInfo.Status.
const (
	ModuleCheckOK     = "ok"
	ModuleCheckWarn   = "warn"
	ModuleCheckNotRun = "not_run"
)

// ModuleCheckSection holds the results of the health checks that ecosystem
// modules contribute through ecosystem.DoctorCheckProvider.
type ModuleCheckSection struct {
	Detected bool              `json:"detected"`
	Checks   []ModuleCheckInfo `json:"checks"`
	Warnings []string          `json:"warnings,omitempty"`
}

// ModuleCheckInfo is the static result of one module doctor check. Status is
// ModuleCheckOK, ModuleCheckWarn or ModuleCheckNotRun (the check has nothing
// the doctor can verify without running a command).
type ModuleCheckInfo struct {
	Module      string `json:"module"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
	// Command is the live verification command the module suggests. The
	// doctor never runs it; it is shown for the user to run.
	Command string `json:"command,omitempty"`
}

// SetModuleCheckSection attaches module doctor check results to the report.
func (r *Report) SetModuleCheckSection(ms *ModuleCheckSection) {
	r.ModuleChecks = ms
}

// ModuleCheckEnv is what static module checks consult. None of it executes
// anything: a check's Command is only resolved on PATH, never run, since a
// cloud CLI would contact the provider and may print a credential.
type ModuleCheckEnv struct {
	// LookupEnv reads the doctor's own environment (os.LookupEnv).
	LookupEnv func(string) (string, bool)
	// Declared holds the variables the project's devenv modules declare,
	// which the devenv shell exports even when the doctor runs outside it.
	Declared map[string]string
	// LookPath resolves an executable name (exec.LookPath).
	LookPath func(string) (string, error)
}

// EvaluateModuleCheck statically evaluates check, contributed by module.
//
// An EnvCheck passes when the variable holds a real value in the project's
// devenv declarations, or in the current environment when the project does
// not declare it (a declaration overrides the environment inside the devenv
// shell). An empty value or a
// placeholder (see cloudcommon.IsUnsetEnvValue) counts as unset, so a
// generated "PLACEHOLDER -- ..." never reports healthy.
//
// A Command is never executed. The doctor only reports whether its
// executable is on PATH and leaves the command for the user to run.
func EvaluateModuleCheck(module string, check ecosystem.DoctorCheck, env ModuleCheckEnv) ModuleCheckInfo {
	info := ModuleCheckInfo{
		Module:      module,
		Name:        check.Name,
		Description: check.Description,
		Command:     check.Command,
	}
	switch {
	case check.EnvCheck != "":
		info.Status, info.Detail = evaluateEnvCheck(check.EnvCheck, env)
	case check.Command != "":
		info.Status, info.Detail = evaluateCommandCheck(check.Command, env)
	default:
		info.Status, info.Detail = ModuleCheckNotRun, "check declares nothing to verify statically"
	}
	return info
}

// evaluateEnvCheck reports whether name holds a real value in the project's
// devenv declarations or, when the project declares none, the current
// environment. A declaration wins because the devenv shell exports it over
// whatever the surrounding shell holds: a real value in the doctor's
// environment does not help when devenv.nix sets a placeholder.
func evaluateEnvCheck(name string, env ModuleCheckEnv) (status, detail string) {
	if v, ok := env.Declared[name]; ok {
		if cloudcommon.IsUnsetEnvValue(v) {
			return ModuleCheckWarn, fmt.Sprintf(
				"%s is declared by the project's devenv modules as a placeholder; set a real value in devenv.local.nix", name)
		}
		return ModuleCheckOK, name + " is declared by the project's devenv modules"
	}
	if env.LookupEnv != nil {
		if v, ok := env.LookupEnv(name); ok && !cloudcommon.IsUnsetEnvValue(v) {
			return ModuleCheckOK, name + " is set in the current environment"
		}
	}
	return ModuleCheckWarn, fmt.Sprintf(
		"%s is unset or a placeholder; set it in devenv.local.nix or the environment", name)
}

// evaluateCommandCheck resolves command's executable on PATH without running
// it.
func evaluateCommandCheck(command string, env ModuleCheckEnv) (status, detail string) {
	fields := strings.Fields(command)
	if len(fields) == 0 || env.LookPath == nil {
		return ModuleCheckNotRun, "not run; verify manually with: " + command
	}
	path, err := env.LookPath(fields[0])
	if err != nil {
		return ModuleCheckWarn, fmt.Sprintf(
			"%s not found on PATH (the devenv shell provides it); then verify with: %s", fields[0], command)
	}
	return ModuleCheckNotRun, fmt.Sprintf(
		"%s found at %s; not run by doctor, verify manually with: %s", fields[0], path, command)
}

// NewModuleCheckSection builds the doctor's module check section. warnings
// carry inputs that could not be read. It returns nil when there is nothing
// to report.
func NewModuleCheckSection(checks []ModuleCheckInfo, warnings []string) *ModuleCheckSection {
	if len(checks) == 0 && len(warnings) == 0 {
		return nil
	}
	return &ModuleCheckSection{Detected: true, Checks: checks, Warnings: warnings}
}

func formatModuleCheckSection(w io.Writer, ms *ModuleCheckSection, okSym, warnSym string) {
	fmt.Fprintln(w, "Ecosystem Checks")
	for _, c := range ms.Checks {
		sym := okSym
		switch c.Status {
		case ModuleCheckWarn:
			sym = warnSym
		case ModuleCheckNotRun:
			sym = "-"
		}
		fmt.Fprintf(w, "  %-20s %s %s\n", c.Name, sym, c.Description)
		if c.Detail != "" {
			fmt.Fprintf(w, "    %s\n", c.Detail)
		}
	}
	for _, warn := range ms.Warnings {
		fmt.Fprintf(w, "  %s %s\n", warnSym, warn)
	}
	fmt.Fprintln(w)
}
