package ecosystem

import "github.com/Quantum-Serendipity/qsdev/internal/sliceutil"

// TaskDefinition represents a standard development task composed of commands
// from one or more ecosystem modules.
type TaskDefinition struct {
	Name        string
	Description string
	Commands    []string
	DependsOn   []string
}

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
	secScan := &TaskDefinition{Name: "security-scan", Description: "Run security scanners"}
	if enabledTools["semgrep"] {
		secScan.Commands = append(secScan.Commands, "semgrep --config auto --error .")
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
