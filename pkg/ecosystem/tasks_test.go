package ecosystem

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAggregateTaskDefinitions_SingleEcosystem(t *testing.T) {
	mod := &MockModule{
		NameVal: "go",
		VerificationCommandsVal: VerificationCommands{
			Build:  []string{"go build ./..."},
			Test:   []string{"go test ./..."},
			Lint:   []string{"go vet ./...", "golangci-lint run"},
			Format: []string{"gofmt -l ."},
		},
	}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{mod},
		staticConfig,
		nil,
	)

	// Should have build, test, lint, format (no typecheck, no security-scan).
	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d: %v", len(tasks), taskNames(tasks))
	}

	assertTaskExists(t, tasks, "build", 1)
	assertTaskExists(t, tasks, "test", 1)
	assertTaskExists(t, tasks, "lint", 2)
	assertTaskExists(t, tasks, "format", 1)

	// Test task should depend on build.
	testTask := findTask(tasks, "test")
	if testTask == nil {
		t.Fatal("test task not found")
		return
	}
	if len(testTask.DependsOn) != 1 || testTask.DependsOn[0] != "build" {
		t.Errorf("test.DependsOn = %v, want [build]", testTask.DependsOn)
	}
}

func TestAggregateTaskDefinitions_MultiEcosystem(t *testing.T) {
	goMod := &MockModule{
		NameVal: "go",
		VerificationCommandsVal: VerificationCommands{
			Build: []string{"go build ./..."},
			Test:  []string{"go test ./..."},
			Lint:  []string{"golangci-lint run"},
		},
	}
	jsMod := &MockModule{
		NameVal: "javascript",
		VerificationCommandsVal: VerificationCommands{
			Build:     []string{"npm run build"},
			Test:      []string{"npm test"},
			Lint:      []string{"npm run lint"},
			Format:    []string{"prettier --check ."},
			TypeCheck: []string{"tsc --noEmit"},
		},
	}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{goMod, jsMod},
		staticConfig,
		nil,
	)

	// Should have build (2), test (2), lint (2), format (1), typecheck (1).
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d: %v", len(tasks), taskNames(tasks))
	}

	assertTaskExists(t, tasks, "build", 2)
	assertTaskExists(t, tasks, "test", 2)
	assertTaskExists(t, tasks, "lint", 2)
	assertTaskExists(t, tasks, "format", 1)
	assertTaskExists(t, tasks, "typecheck", 1)
}

func TestAggregateTaskDefinitions_EmptyFiltered(t *testing.T) {
	// Module with no typecheck commands — typecheck task should be filtered out.
	mod := &MockModule{
		NameVal: "go",
		VerificationCommandsVal: VerificationCommands{
			Build: []string{"go build ./..."},
		},
	}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{mod},
		staticConfig,
		nil,
	)

	for _, task := range tasks {
		if task.Name == "typecheck" {
			t.Error("typecheck task should be filtered out when no module provides typecheck commands")
		}
	}
}

func TestAggregateTaskDefinitions_SecurityScan(t *testing.T) {
	tasks := AggregateTaskDefinitions(
		nil,
		staticConfig,
		map[string]bool{"semgrep": true},
	)

	secTask := findTask(tasks, "security-scan")
	if secTask == nil {
		t.Fatal("security-scan task not found when semgrep is enabled")
		return
	}

	// No SAST-capable module selected: fall back to the baseline rule set,
	// never the registry-chosen, metrics-requiring `--config auto`.
	const want = "semgrep --config p/owasp-top-ten $(if [ -d .semgrep ]; then echo --config .semgrep; fi) --metrics=off --error ."
	if len(secTask.Commands) != 1 || secTask.Commands[0] != want {
		t.Errorf("security-scan commands = %v, want [%s]", secTask.Commands, want)
	}
}

// sastMock adds SemgrepRuleSets to MockModule.
type sastMock struct {
	*MockModule
	ruleSets []string
}

func (m sastMock) SemgrepRuleSets() []string { return m.ruleSets }

func TestAggregateTaskDefinitions_SecurityScanUsesModuleRuleSets(t *testing.T) {
	t.Parallel()

	goMod := sastMock{&MockModule{NameVal: "go"}, []string{"p/golang", "p/owasp-top-ten"}}
	jsMod := sastMock{&MockModule{NameVal: "javascript"}, []string{"p/javascript", "p/owasp-top-ten"}}
	plain := &MockModule{NameVal: "plain"}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{goMod, plain, jsMod},
		staticConfig,
		map[string]bool{"semgrep": true},
	)
	secTask := findTask(tasks, "security-scan")
	if secTask == nil {
		t.Fatal("security-scan task not found when semgrep is enabled")
	}

	const want = "semgrep --config p/golang --config p/javascript --config p/owasp-top-ten $(if [ -d .semgrep ]; then echo --config .semgrep; fi) --metrics=off --error ."
	if len(secTask.Commands) != 1 || secTask.Commands[0] != want {
		t.Errorf("security-scan commands = %v, want [%s]", secTask.Commands, want)
	}
}

func TestAggregateTaskDefinitions_NoSecurityTools(t *testing.T) {
	tasks := AggregateTaskDefinitions(
		nil,
		staticConfig,
		nil,
	)

	secTask := findTask(tasks, "security-scan")
	if secTask != nil {
		t.Error("security-scan task should be omitted when no security tools are enabled")
	}
}

func TestAggregateTaskDefinitions_Dedup(t *testing.T) {
	mod1 := &MockModule{
		NameVal:                 "a",
		VerificationCommandsVal: VerificationCommands{Test: []string{"shared-test-cmd"}},
	}
	mod2 := &MockModule{
		NameVal:                 "b",
		VerificationCommandsVal: VerificationCommands{Test: []string{"shared-test-cmd"}},
	}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{mod1, mod2},
		staticConfig,
		nil,
	)

	testTask := findTask(tasks, "test")
	if testTask == nil {
		t.Fatal("test task not found")
		return
	}

	if len(testTask.Commands) != 1 {
		t.Errorf("expected 1 command after dedup, got %d: %v", len(testTask.Commands), testTask.Commands)
	}
}

func TestAggregateTaskDefinitions_StableOrder(t *testing.T) {
	mod := &MockModule{
		NameVal: "full",
		VerificationCommandsVal: VerificationCommands{
			Build:     []string{"build-cmd"},
			Test:      []string{"test-cmd"},
			Lint:      []string{"lint-cmd"},
			Format:    []string{"format-cmd"},
			TypeCheck: []string{"typecheck-cmd"},
		},
	}

	tasks := AggregateTaskDefinitions(
		[]EcosystemModule{mod},
		staticConfig,
		map[string]bool{"semgrep": true, "gitleaks": true},
	)

	expectedOrder := []string{"build", "test", "lint", "format", "typecheck", "security-scan"}
	if len(tasks) != len(expectedOrder) {
		t.Fatalf("expected %d tasks, got %d: %v", len(expectedOrder), len(tasks), taskNames(tasks))
	}

	for i, want := range expectedOrder {
		if tasks[i].Name != want {
			t.Errorf("task[%d] = %q, want %q", i, tasks[i].Name, want)
		}
	}
}

func TestAggregateTaskDefinitions_SecurityScanBothTools(t *testing.T) {
	tasks := AggregateTaskDefinitions(
		nil,
		staticConfig,
		map[string]bool{"semgrep": true, "gitleaks": true},
	)

	secTask := findTask(tasks, "security-scan")
	if secTask == nil {
		t.Fatal("security-scan task not found")
		return
	}

	if len(secTask.Commands) != 2 {
		t.Errorf("expected 2 security-scan commands, got %d: %v", len(secTask.Commands), secTask.Commands)
	}
}

// TestAggregateTaskDefinitions_SecurityScanOpengrep is the F217 regression:
// enabling opengrep must actually run it, against the rule library delivered
// into the project, failing the task on findings.
func TestAggregateTaskDefinitions_SecurityScanOpengrep(t *testing.T) {
	t.Parallel()

	const opengrepCmd = "opengrep scan --config .opengrep/rules/core --error"
	tests := []struct {
		name    string
		enabled map[string]bool
		want    []string
	}{
		{name: "opengrep only", enabled: map[string]bool{"opengrep": true}, want: []string{opengrepCmd}},
		{
			name:    "opengrep with gitleaks",
			enabled: map[string]bool{"opengrep": true, "gitleaks": true},
			want:    []string{opengrepCmd, "gitleaks detect --no-banner"},
		},
		{name: "opengrep disabled", enabled: map[string]bool{"opengrep": false}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			secTask := findTask(AggregateTaskDefinitions(nil, staticConfig, tt.enabled), "security-scan")
			var got []string
			if secTask != nil {
				got = secTask.Commands
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("security-scan commands = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSemgrepScanCommand_LocalRules is the F202 regression: the security-scan
// task runs the project's own rules from .semgrep/ alongside the ecosystem
// packs when that directory exists, and scans cleanly when it does not. The
// command is executed by bash (as the devenv task script runs it, under
// errexit and nounset) against a stub semgrep that records its arguments.
func TestSemgrepScanCommand_LocalRules(t *testing.T) {
	t.Parallel()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}

	goMod := sastMock{&MockModule{NameVal: "go"}, []string{"p/golang"}}
	cmd := semgrepScanCommand([]EcosystemModule{goMod})

	tests := []struct {
		name     string
		localDir bool
		want     string
	}{
		{name: "no local rules", want: "--config p/golang --metrics=off --error ."},
		{name: "local rules dir", localDir: true, want: "--config p/golang --config .semgrep --metrics=off --error ."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := t.TempDir()
			if tt.localDir {
				if err := os.Mkdir(filepath.Join(project, SemgrepLocalRulesDir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			bin := t.TempDir()
			stub := "#!" + bash + "\nprintf '%s\\n' \"$*\"\n"
			if err := os.WriteFile(filepath.Join(bin, "semgrep"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}

			run := exec.Command(bash, "-c", "set -euo pipefail\n"+cmd)
			run.Dir = project
			run.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("running %q: %v\n%s", cmd, err, out)
			}
			if got := strings.TrimSpace(string(out)); got != tt.want {
				t.Errorf("semgrep args = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- helpers ---

func findTask(tasks []TaskDefinition, name string) *TaskDefinition {
	for i := range tasks {
		if tasks[i].Name == name {
			return &tasks[i]
		}
	}
	return nil
}

func taskNames(tasks []TaskDefinition) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.Name
	}
	return names
}

func assertTaskExists(t *testing.T, tasks []TaskDefinition, name string, wantCmds int) {
	t.Helper()
	task := findTask(tasks, name)
	if task == nil {
		t.Errorf("task %q not found", name)
		return
	}
	if len(task.Commands) != wantCmds {
		t.Errorf("task %q has %d commands, want %d: %v", name, len(task.Commands), wantCmds, task.Commands)
	}
}
