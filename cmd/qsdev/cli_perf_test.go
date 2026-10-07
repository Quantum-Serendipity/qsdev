//go:build perf && linux

package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/perfbudget"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// cliRow is one command TestCLIPerf measures.
type cliRow struct {
	name   string   // the invocation, as rows and skips name it
	budget string   // the budgets.yaml id it is held to
	argv   []string // run in the project, after any network namespace prefix
}

// TestCLIPerf measures qsdev's start-up commands, every invocation declared
// read-only (cmdutil.MarkReadOnly, so a new read-only command is measured
// without touching this test), and init creating and joining a project, in
// a hermetic generated project, and holds each to its budget in
// budgets.yaml. Each runs without network access and with procexec's
// forbid-exec guard on: it must not trip the guard or panic, and it must
// exit as it does with the network reachable, so no command depends on the
// network.
func TestCLIPerf(t *testing.T) {
	s := newCLIPerfSuite(t)
	for _, row := range s.rows() {
		t.Run(row.name, func(t *testing.T) {
			spec := perfbudget.Spec{Argv: append(slices.Clone(s.netns), row.argv...), Env: s.env, Dir: s.project}
			stats := s.budgets.Run(t, spec, s.budgets.Runs)
			factor := s.budgets.Calibrate(t, s.noop, s.calEnv, perfbudget.RecalibrationRuns)
			checkCLIRun(t, row.name+" without network", stats)

			// One run with the network reachable, still under forbid-exec and
			// with the update check off, so nothing it starts reaches out.
			once := s.budgets
			once.Warmups = 0
			online := once.Run(t, perfbudget.Spec{Argv: row.argv, Env: s.env, Dir: s.project}, 1)
			checkCLIRun(t, row.name+" with network", online)
			if online.ExitCode != stats.ExitCode {
				t.Errorf("qsdev %s exits %d without network but %d with it, so it depends on the network\nstderr without network:\n%s",
					row.name, stats.ExitCode, online.ExitCode, stats.Stderr)
			}
			s.budgets.Check(t, perfbudget.Row{Budget: row.budget, Subject: "qsdev " + row.name}, stats, factor)
		})
	}
}

// cliPerfSuite is the fixture TestCLIPerf measures in: the budgets, the
// qsdev binary under test, the calibration program, the network namespace
// prefix and a committed, initialized project with clean sources for init
// to create and join from.
type cliPerfSuite struct {
	budgets perfbudget.Budgets
	bin     string
	noop    string
	netns   []string
	calEnv  []string
	env     []string // the measured commands': hermetic, forbid-exec on
	project string   // initialized and committed; read-only commands run here
	create  string   // a Go project with no qsdev configuration
	join    string   // a clean clone of project
}

func newCLIPerfSuite(t *testing.T) *cliPerfSuite {
	t.Helper()
	b, err := perfbudget.Load()
	if err != nil {
		t.Fatalf("loading budgets: %v", err)
	}
	root := t.TempDir()
	bin, err := perfbudget.Build(filepath.Join(root, "bin"))
	if err != nil {
		t.Fatal(err)
	}
	noop, err := perfbudget.BuildNoop(filepath.Join(root, "noop"))
	if err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Dir(bin)
	calEnv, err := perfbudget.IsolatedEnv(filepath.Join(root, "calibration"), binDir)
	if err != nil {
		t.Fatal(err)
	}
	userEnv, err := perfbudget.IsolatedEnv(filepath.Join(root, "user"), binDir)
	if err != nil {
		t.Fatal(err)
	}
	s := &cliPerfSuite{
		budgets: b, bin: bin, noop: noop, netns: perfbudget.NetNS(t), calEnv: calEnv,
		project: filepath.Join(root, "project"),
		create:  filepath.Join(root, "create-source"),
		join:    filepath.Join(root, "join-source"),
	}
	// Setup and the update check stay off in every run, so neither the
	// fixture nor a run with network reaches out.
	env := append(slices.Clone(userEnv), "QSDEV_SKIP_SETUP=1", branding.Get().EnvNoUpdate+"=1")

	s.goRepo(t, s.project, env)
	perfbudget.RunIn(t, s.project, env, append(slices.Clone(s.netns), bin, "init", "--yes")...)
	perfbudget.RunIn(t, s.project, env, "git", "add", "-A")
	perfbudget.RunIn(t, s.project, env, "git", "-c", "core.hooksPath=/dev/null", "-c", "user.name=perf",
		"-c", "user.email=perf@example.invalid", "commit", "-q", "-m", "qsdev init")
	perfbudget.RunIn(t, root, env, "git", "clone", "-q", s.project, s.join)
	s.goRepo(t, s.create, env)

	s.env = append(env, procexec.ForbidExecEnv+"=1")
	return s
}

// goRepo writes the perfbudget Go project into dir as a fresh git
// repository.
func (s *cliPerfSuite) goRepo(t *testing.T, dir string, env []string) {
	t.Helper()
	if err := perfbudget.WriteGoProject(dir); err != nil {
		t.Fatal(err)
	}
	perfbudget.RunIn(t, dir, env, "git", "init", "-q")
}

// rows lists the measured commands: the start-up commands, every read-only
// invocation in the command tree, and init creating a project and joining
// one.
func (s *cliPerfSuite) rows() []cliRow {
	var rows []cliRow
	for _, args := range [][]string{{"version"}, {"--help"}, {"completion", "zsh"}, {"completion", "bash"}} {
		rows = append(rows, cliRow{name: strings.Join(args, " "), budget: "cli-startup", argv: append([]string{s.bin}, args...)})
	}
	for _, inv := range readOnlyInvocations(instance.NewRootCommand()) {
		rows = append(rows, cliRow{name: inv, budget: readOnlyBudget(inv), argv: append([]string{s.bin}, strings.Fields(inv)...)})
	}
	return append(rows,
		cliRow{name: "init --yes (create)", budget: "cli-init", argv: s.initInCopy(s.create)},
		cliRow{name: "init --yes (join)", budget: "cli-init", argv: s.initInCopy(s.join)},
	)
}

// readOnlyBudget names the budget a read-only invocation is held to: the
// MCP diagnostics, which inspect every configured server, have their own.
func readOnlyBudget(inv string) string {
	if inv == "mcp" || strings.HasPrefix(inv, "mcp ") {
		return "cli-mcp-status"
	}
	return "cli-readonly"
}

// initInCopy runs `qsdev init --yes` in a fresh copy of source, so every
// timed run starts from the same tree. The shell execs qsdev, so the run's
// CPU is qsdev's plus the mktemp and cp it waited for, a few milliseconds
// against the init budget.
func (s *cliPerfSuite) initInCopy(source string) []string {
	const script = `d=$(mktemp -d) && cp -R "$1"/. "$d" && cd "$d" && exec "$2" init --yes`
	return []string{"sh", "-c", script, "sh", source, s.bin}
}

// checkCLIRun fails t when a run tripped procexec's forbid-exec guard or
// otherwise panicked, or when its invocation does not exist.
func checkCLIRun(t *testing.T, what string, s perfbudget.Stats) {
	t.Helper()
	switch {
	case strings.Contains(s.Stderr, "exec forbidden under "+procexec.ForbidExecEnv):
		t.Errorf("qsdev %s ran a forbidden exec:\n%s", what, s.Stderr)
	case strings.Contains(s.Stderr, "panic:"):
		t.Errorf("qsdev %s panicked:\n%s", what, s.Stderr)
	case strings.Contains(s.Stderr, "unknown flag") || strings.Contains(s.Stderr, "unknown command"):
		t.Errorf("qsdev %s is not a valid invocation:\n%s", what, s.Stderr)
	}
}
