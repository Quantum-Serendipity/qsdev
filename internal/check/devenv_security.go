package check

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// CheckDevenvSecurityFloor reports whether the project's devenv modules still
// enable every security git hook and strip every credential variable the
// devenv.nix generated for the project does (ctx.ExpectedDevenvHooks,
// ctx.ExpectedUnsetVars), and define those hooks as it does
// (ctx.ExpectedDevenvHookSettings), so a hook cannot be neutralised by
// replacing its entry or excluding every file. devenv.nix is human-edited,
// so the generated-file checks do not compare it; this check holds the
// hand-edited file to the generated security floor while allowing anything
// else it adds. A deleted devenv.nix enables nothing and fails, as does a
// module that cannot be read or verified (one that imports another module,
// for example).
func CheckDevenvSecurityFloor(ctx CheckContext) CheckResult {
	result := CheckResult{
		Category: CategorySecurityHarden,
		Name:     "devenv_security_floor",
		Severity: SeverityInfo,
		FilePath: toolreg.DevenvNixFile,
	}
	switch {
	case ctx.ExpectedGenerationErr != nil:
		result.Status, result.Severity = StatusWarn, SeverityMedium
		result.Message = "Cannot verify the git hooks and stripped variables of " + toolreg.DevenvNixFile +
			": the files qsdev generates for this project are unknown"
		return result
	case ctx.ExpectedDevenvHooks == nil && ctx.ExpectedUnsetVars == nil:
		result.Status = StatusSkip
		result.Message = "No " + toolreg.DevenvNixFile + " is generated for this project"
		return result
	}

	var problems []string
	if ctx.DevenvSecurityErr != nil {
		// What the modules declare is unknown, so listing what is missing
		// would only repeat everything expected.
		problems = append(problems, fmt.Sprintf("cannot verify the devenv modules: %v", ctx.DevenvSecurityErr))
	} else {
		problems = devenvFloorProblems(ctx, &result)
	}
	if len(problems) == 0 {
		result.Status = StatusPass
		result.Message = fmt.Sprintf("%s enables every generated security git hook (%d) and strips every generated credential variable (%d)",
			toolreg.DevenvNixFile, len(ctx.ExpectedDevenvHooks), len(ctx.ExpectedUnsetVars))
		return result
	}
	result.Status, result.Severity = StatusFail, SeverityHigh
	result.Message = toolreg.DevenvNixFile + " is weaker than the generated security floor: " + strings.Join(problems, "; ")
	// devenv.nix only fails here after a hand edit, and 'init --update' leaves a
	// hand-edited devenv.nix in place (it writes a sidecar), so name the step
	// that overwrites it.
	result.Remediation = restoreGeneratedRemediation(toolreg.DevenvNixFile) +
		" (this discards your edits; check your user/org defaults file too); keep local changes in devenv.local.nix without disabling these"
	return result
}

// devenvFloorProblems returns what the project's devenv modules no longer
// enable, strip or define as generated, recording each list in
// result.Metadata.
func devenvFloorProblems(ctx CheckContext, result *CheckResult) []string {
	var problems []string
	meta := map[string]string{}
	report := func(key, what string, items []string) {
		if len(items) > 0 {
			problems = append(problems, what+": "+strings.Join(items, ", "))
			meta[key] = strings.Join(items, ",")
		}
	}
	report("missing_hooks", "security git hooks not enabled", missingFrom(ctx.ExpectedDevenvHooks, ctx.DevenvHooks))
	report("missing_unset_vars", "credential variables not stripped (unsetEnvVars)", missingFrom(ctx.ExpectedUnsetVars, ctx.DevenvUnsetVars))
	report("changed_hook_settings", "security git hook settings differ from the generated file",
		changedSettings(ctx.ExpectedDevenvHookSettings, ctx.DevenvHookSettings))
	if len(meta) > 0 {
		result.Metadata = meta
	}
	return problems
}

// missingFrom returns the members of want that have does not contain.
func missingFrom(want, have []string) []string {
	var missing []string
	for _, w := range want {
		if !slices.Contains(have, w) {
			missing = append(missing, w)
		}
	}
	return missing
}

// changedSettings returns, sorted, the attribute paths set in want or have
// whose values differ (including a path only one of them sets).
func changedSettings(want, have map[string]string) []string {
	var changed []string
	for _, path := range slices.Sorted(maps.Keys(want)) {
		if v, ok := have[path]; !ok || v != want[path] {
			changed = append(changed, path)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(have)) {
		if _, ok := want[path]; !ok {
			changed = append(changed, path)
		}
	}
	slices.Sort(changed)
	return changed
}
