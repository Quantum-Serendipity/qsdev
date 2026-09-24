package ecosystem

import (
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/rules"
)

// TaskDefinition represents a standard development task composed of commands
// from one or more ecosystem modules.
type TaskDefinition struct {
	Name        string
	Description string
	Commands    []string
	DependsOn   []string
}

// SecurityScanTask is the name of the task that runs the enabled security
// scanners (semgrep, opengrep, gitleaks).
const SecurityScanTask = "security-scan"

// TaskScriptPrefix is prepended to a task name to form the name of the devenv
// script that runs it (e.g. "qsdev-security-scan"). The devenv addon renders
// the scripts under these names and posture inspects them by it.
const TaskScriptPrefix = "qsdev-"

// SemgrepLocalRulesDir is the project-relative directory holding the project's
// own Semgrep rules. The security-scan task passes it to semgrep alongside the
// ecosystem rule packs whenever it exists, so teams can add custom rules
// without qsdev overwriting them.
const SemgrepLocalRulesDir = ".semgrep"

// AggregateTaskDefinitions builds standard development tasks from ecosystem modules.
// Standard tasks: build, test, lint, format, typecheck, security-scan.
// Empty tasks (no commands) are filtered out.
func AggregateTaskDefinitions(
	modules []EcosystemModule,
	configFor func(EcosystemModule) ModuleConfig,
	enabledTools map[string]bool,
) []TaskDefinition {
	tasks := map[string]*TaskDefinition{
		"build":     {Name: "build", Description: "Build all projects"},
		"test":      {Name: "test", Description: "Run all test suites", DependsOn: []string{"build"}},
		"lint":      {Name: "lint", Description: "Run all linters"},
		"format":    {Name: "format", Description: "Format all source code"},
		"typecheck": {Name: "typecheck", Description: "Run type checkers"},
	}

	for _, mod := range modules {
		vc := mod.VerificationCommands(configFor(mod))
		tasks["build"].Commands = append(tasks["build"].Commands, vc.Build...)
		tasks["test"].Commands = append(tasks["test"].Commands, vc.Test...)
		tasks["lint"].Commands = append(tasks["lint"].Commands, vc.Lint...)
		tasks["format"].Commands = append(tasks["format"].Commands, vc.Format...)
		tasks["typecheck"].Commands = append(tasks["typecheck"].Commands, vc.TypeCheck...)
	}

	// Security-scan from enabled tools.
	secScan := &TaskDefinition{Name: SecurityScanTask, Description: "Run security scanners"}
	if enabledTools["semgrep"] {
		secScan.Commands = append(secScan.Commands, semgrepScanCommand(modules))
	}
	if enabledTools["opengrep"] {
		secScan.Commands = append(secScan.Commands, opengrepScanCommand)
	}
	if enabledTools["gitleaks"] {
		secScan.Commands = append(secScan.Commands, "gitleaks detect --no-banner")
	}

	// Collect non-empty tasks in stable order.
	order := []string{"build", "test", "lint", "format", "typecheck"}
	var result []TaskDefinition
	for _, name := range order {
		t := tasks[name]
		t.Commands = sliceutil.Dedup(t.Commands)
		if len(t.Commands) > 0 {
			result = append(result, *t)
		}
	}

	// Add security-scan if non-empty.
	if len(secScan.Commands) > 0 {
		result = append(result, *secScan)
	}

	return result
}

// opengrepScanCommand runs OpenGrep against the core taint-rule library the
// opengrep tool delivers into the project, failing the task on any finding.
// OpenGrep has no project config file, so the rules are passed directly.
const opengrepScanCommand = "opengrep scan --config " + rules.ProjectCoreDir + " --error"

// defaultSemgrepRuleSet is scanned when no selected module declares rule sets.
const defaultSemgrepRuleSet = "p/owasp-top-ten"

// semgrepLocalRulesArg expands, in the task's shell, to a --config flag for
// SemgrepLocalRulesDir when that directory exists and to nothing otherwise,
// so a project without custom rules still scans cleanly.
const semgrepLocalRulesArg = "$(if [ -d " + SemgrepLocalRulesDir + " ]; then echo --config " + SemgrepLocalRulesDir + "; fi)"

// semgrepScanCommand builds the semgrep invocation from the rule sets the
// selected modules declare via SASTModule, one --config flag per set (sorted
// and deduplicated), plus the project's local rules in SemgrepLocalRulesDir
// when present. This scans the project's ecosystem rules rather than
// `--config auto`, which lets the registry pick rules and requires metrics to
// be enabled; explicit rule sets allow --metrics=off. Path exclusions come
// from the .semgrepignore the semgrep tool generates, which semgrep reads from
// the scan root.
func semgrepScanCommand(modules []EcosystemModule) string {
	var ruleSets []string
	for _, mod := range modules {
		if sast, ok := mod.(SASTModule); ok {
			ruleSets = append(ruleSets, sast.SemgrepRuleSets()...)
		}
	}
	slices.Sort(ruleSets)
	ruleSets = slices.Compact(ruleSets)
	if len(ruleSets) == 0 {
		ruleSets = []string{defaultSemgrepRuleSet}
	}

	var b strings.Builder
	b.WriteString("semgrep")
	for _, rs := range ruleSets {
		b.WriteString(" --config ")
		b.WriteString(rs)
	}
	b.WriteString(" ")
	b.WriteString(semgrepLocalRulesArg)
	b.WriteString(" --metrics=off --error .")
	return b.String()
}
