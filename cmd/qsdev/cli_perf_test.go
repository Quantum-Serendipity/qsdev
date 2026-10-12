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
// budgets.yaml. Each runs on a hermetic PATH, where every host tool qsdev
// probes is an empty stub (see hermeticPath), so a row measures qsdev's own
// CPU and not that of whatever tools the machine has installed. Each runs
// without network access and with procexec's
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

// hostTools are the host tools the measured commands genuinely need, which
// run as themselves: the project is a git repository and qsdev reads it
// through git (status and repair resolve its hooks path), which a stub
// would answer for no repository at all. Every other tool they probe is a
// stub (see probedTools).
var hostTools = []string{"git"}

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
	// copyTools are the absolute sh, mktemp and cp initInCopy runs: the
	// measured PATH holds none of them.
	copyTools []string
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
	s := &cliPerfSuite{
		budgets: b, bin: bin, noop: noop, netns: perfbudget.NetNS(t),
		project:   filepath.Join(root, "project"),
		create:    filepath.Join(root, "create-source"),
		join:      filepath.Join(root, "join-source"),
		copyTools: lookPaths(t, "sh", "mktemp", "cp"),
	}
	// The fixture runs on a hermetic PATH too, so the project it creates
	// does not depend on what the host has installed. Before the project
	// exists, its stubs are the tools doctor's registry lists.
	setupPath := s.hermeticPath(t, filepath.Join(root, "path-setup"), "")
	s.calEnv = s.isolatedEnv(t, filepath.Join(root, "calibration"), setupPath)
	env := s.isolatedEnv(t, filepath.Join(root, "user"), setupPath)

	s.goRepo(t, s.project, env)
	perfbudget.RunIn(t, s.project, env, append(slices.Clone(s.netns), bin, "init", "--yes")...)
	perfbudget.RunIn(t, s.project, env, "git", "add", "-A")
	perfbudget.RunIn(t, s.project, env, "git", "-c", "core.hooksPath=/dev/null", "-c", "user.name=perf",
		"-c", "user.email=perf@example.invalid", "commit", "-q", "-m", "qsdev init")
	perfbudget.RunIn(t, root, env, "git", "clone", "-q", s.project, s.join)
	s.goRepo(t, s.create, env)

	// The measured commands also find a stub for every program the
	// project's hooks run, which doctor probes there.
	measuredPath := s.hermeticPath(t, filepath.Join(root, "path"), s.project)
	s.env = append(s.isolatedEnv(t, filepath.Join(root, "user"), measuredPath), procexec.ForbidExecEnv+"=1")
	return s
}

// isolatedEnv returns the hermetic environment under root with PATH path,
// setup and the update check off, so neither the fixture nor a run with
// network reaches out.
func (s *cliPerfSuite) isolatedEnv(t *testing.T, root, path string) []string {
	t.Helper()
	env, err := perfbudget.IsolatedEnv(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return append(env, "QSDEV_SKIP_SETUP=1", branding.Get().EnvNoUpdate+"=1")
}

// hermeticPath creates, under dir, the hermetic PATH of a command run in
// project ("" before it exists): the qsdev under test, hostTools, and a
// stub for every other tool qsdev probes there (see probedTools).
func (s *cliPerfSuite) hermeticPath(t *testing.T, dir, project string) string {
	t.Helper()
	probed, err := probedTools(project)
	if err != nil {
		t.Fatal(err)
	}
	path, err := perfbudget.HermeticPath(dir, perfbudget.HermeticTools{
		Bin: filepath.Dir(s.bin), Host: hostTools, Probed: probed, Stub: s.noop,
	})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// lookPaths resolves each name on the test's PATH.
func lookPaths(t *testing.T, names ...string) []string {
	t.Helper()
	paths := make([]string, 0, len(names))
	for _, name := range names {
		p, err := procexec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
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
// against the init budget. The shell, mktemp and cp are absolute (see
// copyTools).
func (s *cliPerfSuite) initInCopy(source string) []string {
	const script = `d=$("$3" -d) && "$4" -R "$1"/. "$d" && cd "$d" && exec "$2" init --yes`
	sh, mktemp, cp := s.copyTools[0], s.copyTools[1], s.copyTools[2]
	return []string{sh, "-c", script, "sh", source, s.bin, mktemp, cp}
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
