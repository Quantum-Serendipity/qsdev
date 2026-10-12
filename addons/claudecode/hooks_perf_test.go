//go:build perf && linux

package claudecode

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/perfbudget"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestHookPerf measures every command hook a generated project registers,
// run as Claude Code runs it, and holds it to budgets.yaml. The hooks come
// only from the generated settings.json (claudesettings.HooksFor), so a hook
// added to the registry is measured without touching this test.
//
// The rows subtest holds each (hook, payload) pair of the full-preset project
// to its budget; the aggregate subtest holds one tool call, the standard
// project's and the full project's, to its critical-path and total budgets.
func TestHookPerf(t *testing.T) {
	s := newHookPerfSuite(t)
	full := s.project(t, "full", "--tier", "full", "--claude-hooks", strings.Join(SelectableHookPresets(), ","))
	standard := s.project(t, "standard")

	t.Run("rows", func(t *testing.T) {
		rows := 0
		for _, p := range perfbudget.Payloads() {
			hooks := full.commandHooks(p.Event, p.Tool)
			if len(hooks) == 0 {
				t.Errorf("no generated hook runs for payload %s (%s %s); the corpus or the generated hooks lost coverage", p.Name, p.Event, p.Tool)
			}
			for _, h := range hooks {
				rows++
				t.Run(h.subject+"/"+p.Name, func(t *testing.T) {
					t.Parallel()
					m := s.measureHook(t, full, h, p)
					row := perfbudget.Row{Budget: hookBudget(h, p, m.Stats), Subject: h.subject, Payload: p.Name}
					s.budgets.Check(t, row, m.Stats, m.factor)
				})
			}
		}
		t.Logf("measured %d (hook, payload) rows", rows)
	})

	t.Run("aggregate", func(t *testing.T) {
		for _, tt := range []struct {
			proj            *hookPerfProject
			payload         string
			critical, total string
		}{
			{standard, "bash_ls", "toolcall-critical-bash-standard", ""},
			{full, "bash_ls", "toolcall-critical-bash-full", "toolcall-total-bash-full"},
			{full, "edit_2k", "toolcall-critical-edit-full", ""},
			{full, "read", "toolcall-critical-read-full", ""},
		} {
			s.checkToolCall(t, tt.proj, payloadNamed(t, tt.payload), tt.critical, tt.total)
		}
	})
}

// hookBudget names the budget a hook's row is held to: an in-binary hook by
// payload size, a hook script by whether it decided the call (denied with
// exit 2, or answered Claude Code on stdout) or let it through untouched.
func hookBudget(h perfHook, p perfbudget.Payload, s perfbudget.Stats) string {
	switch {
	case h.inBinary && p.Large:
		return "hook-inbinary-large"
	case h.inBinary:
		return "hook-inbinary-small"
	case s.ExitCode == 2 || strings.TrimSpace(s.Stdout) != "":
		return "hook-script-hit"
	default:
		return "hook-script-miss"
	}
}

// checkToolCall holds one call of p.Tool in proj to its budgets. Claude Code
// starts every matching hook at once, so the call's critical path is the
// largest CPU p95 among them and its cost is their sum. The call is held at
// the mean of its hooks' calibration factors.
func (s *hookPerfSuite) checkToolCall(t *testing.T, proj *hookPerfProject, p perfbudget.Payload, critical, total string) {
	t.Helper()
	hooks := proj.commandHooks(p.Event, p.Tool)
	if len(hooks) == 0 {
		t.Errorf("%s project: no generated hook runs for %s", proj.name, p.Tool)
		return
	}
	stats := make([]perfbudget.Stats, 0, len(hooks))
	names := make([]string, 0, len(hooks))
	factor := 0.0
	for _, h := range hooks {
		m := s.measureHook(t, proj, h, p)
		stats = append(stats, m.Stats)
		names = append(names, h.subject)
		factor += m.factor / float64(len(hooks))
	}
	subject := p.Tool + " call, " + proj.name + " (" + strings.Join(names, " + ") + ")"
	t.Run(critical, func(t *testing.T) {
		s.budgets.Check(t, perfbudget.Row{Budget: critical, Subject: subject, Payload: p.Name}, combine(stats, longer), factor)
	})
	if total != "" {
		t.Run(total, func(t *testing.T) {
			s.budgets.Check(t, perfbudget.Row{Budget: total, Subject: subject, Payload: p.Name}, combine(stats, sum), factor)
		})
	}
}

func sum(a, b time.Duration) time.Duration    { return a + b }
func longer(a, b time.Duration) time.Duration { return max(a, b) }

// combine folds per-hook statistics into one tool call's, statistic by
// statistic.
func combine(stats []perfbudget.Stats, fold func(a, b time.Duration) time.Duration) perfbudget.Stats {
	out := stats[0]
	for _, s := range stats[1:] {
		out.N = min(out.N, s.N)
		out.ExitCode = max(out.ExitCode, s.ExitCode)
		out.CPU = foldDist(out.CPU, s.CPU, fold)
		out.Wall = foldDist(out.Wall, s.Wall, fold)
	}
	return out
}

func foldDist(a, b perfbudget.Dist, fold func(a, b time.Duration) time.Duration) perfbudget.Dist {
	return perfbudget.Dist{P50: fold(a.P50, b.P50), P95: fold(a.P95, b.P95), Max: fold(a.Max, b.Max)}
}

// hookPerfSuite is what TestHookPerf's subtests share: the budgets, the
// qsdev binary under test, the network namespace prefix, a hermetic
// environment to calibrate in, and the measurements taken so far.
type hookPerfSuite struct {
	budgets perfbudget.Budgets
	root    string
	bin     string
	noop    string
	netns   []string
	calEnv  []string
	// slots bounds how many rows measure at once. A row's CPU time is its
	// own processes' whatever runs beside it, and each row recalibrates
	// under the same load, so measuring rows side by side keeps the suite
	// within the CI step's time without changing what it holds them to.
	// Half the CPUs leaves room for the CLI suite running beside it.
	slots chan struct{}

	mu      sync.Mutex
	results map[string]measured
}

// measured is a measurement and the calibration factor taken next to it.
type measured struct {
	perfbudget.Stats
	factor float64
}

func newHookPerfSuite(t *testing.T) *hookPerfSuite {
	t.Helper()
	b, err := perfbudget.Load()
	if err != nil {
		t.Fatalf("loading budgets: %v", err)
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	bin, err := perfbudget.Build(binDir)
	if err != nil {
		t.Fatal(err)
	}
	noop, err := perfbudget.BuildNoop(filepath.Join(root, "noop"))
	if err != nil {
		t.Fatal(err)
	}
	calEnv, err := perfbudget.IsolatedEnv(filepath.Join(root, "calibration"), perfbudget.HostPath(binDir))
	if err != nil {
		t.Fatal(err)
	}
	return &hookPerfSuite{
		budgets: b, root: root, bin: bin, noop: noop, netns: perfbudget.NetNS(t), calEnv: calEnv,
		slots: make(chan struct{}, max(1, runtime.NumCPU()/2)), results: map[string]measured{},
	}
}

// hookPerfProject is a generated project with its own git repository, home,
// temp and XDG directories.
type hookPerfProject struct {
	name     string
	dir      string
	env      []string // as Claude Code runs hooks: hermetic, settings env and CLAUDE_PROJECT_DIR set
	settings claudesettings.Settings
}

// project creates a Go project in a fresh git repository, runs
// `qsdev init --yes <initArgs>` in it without network access, and parses
// the settings.json it wrote.
func (s *hookPerfSuite) project(t *testing.T, name string, initArgs ...string) *hookPerfProject {
	t.Helper()
	root := filepath.Join(s.root, "project-"+name)
	dir := filepath.Join(root, "project")
	// A hook script is host software: its interpreter and the tools it runs
	// are part of what it costs, so it runs on the host's PATH.
	env, err := perfbudget.IsolatedEnv(filepath.Join(root, "user"), perfbudget.HostPath(filepath.Dir(s.bin)))
	if err != nil {
		t.Fatal(err)
	}
	if err := perfbudget.WriteGoProject(dir); err != nil {
		t.Fatal(err)
	}
	perfbudget.RunIn(t, dir, env, "git", "init", "-q")
	initArgv := append(slices.Clone(s.netns), append([]string{s.bin, "init", "--yes"}, initArgs...)...)
	perfbudget.RunIn(t, dir, append(slices.Clone(env), "QSDEV_SKIP_SETUP=1"), initArgv...)
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(claudesettings.ProjectRelPath)))
	if err != nil {
		t.Fatalf("%s project: %v", name, err)
	}
	settings, err := claudesettings.Parse(data)
	if err != nil {
		t.Fatalf("%s project: parsing settings.json: %v", name, err)
	}
	for k, v := range settings.Env {
		env = append(env, k+"="+v)
	}
	return &hookPerfProject{name: name, dir: dir, env: append(env, "CLAUDE_PROJECT_DIR="+dir), settings: settings}
}

// perfHook is a generated command hook and the name rows give it.
type perfHook struct {
	claudesettings.Hook
	// subject names the hook as budgets.yaml skips do: its script's name
	// without extension, or the binary and subcommand of an in-binary hook.
	subject string
	// inBinary marks a hook the qsdev binary itself serves.
	inBinary bool
}

// commandHooks returns the command hooks Claude Code runs for a call of
// tool at event, as the generated settings.json registers them.
func (p *hookPerfProject) commandHooks(event, tool string) []perfHook {
	var out []perfHook
	for _, h := range p.settings.HooksFor(event, tool) {
		if h.Type == claudesettings.HookTypeCommand {
			out = append(out, newPerfHook(h))
		}
	}
	return out
}

// newPerfHook names h from its command's first word: the script it runs
// (`"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py` is
// package-guard) or, for the binary the branding names, the binary and its
// subcommand (`qsdev selfprotect`).
func newPerfHook(h claudesettings.Hook) perfHook {
	fields := strings.Fields(h.Command)
	if len(fields) == 0 {
		return perfHook{Hook: h, subject: h.Command}
	}
	base := path.Base(strings.ReplaceAll(fields[0], `"`, ""))
	if app := branding.Get().AppName; base == app && len(fields) > 1 {
		return perfHook{Hook: h, subject: app + " " + fields[1], inBinary: true}
	}
	return perfHook{Hook: h, subject: strings.TrimSuffix(base, path.Ext(base))}
}

// measureHook runs h as Claude Code does, `sh -c <command>` in the project
// with the payload on stdin, without network access, then recalibrates. A
// (project, hook, payload) triple is measured once per test; the rows that
// measure concurrently are all distinct triples.
func (s *hookPerfSuite) measureHook(t *testing.T, proj *hookPerfProject, h perfHook, p perfbudget.Payload) measured {
	t.Helper()
	key := proj.name + "\x00" + h.Command + "\x00" + p.Name
	s.mu.Lock()
	m, ok := s.results[key]
	s.mu.Unlock()
	if ok {
		return m
	}
	stdin, err := p.JSON(proj.dir)
	if err != nil {
		t.Fatal(err)
	}
	s.slots <- struct{}{}
	defer func() { <-s.slots }()
	m = measured{Stats: s.budgets.Run(t, perfbudget.Spec{
		Argv:  append(slices.Clone(s.netns), "sh", "-c", h.Command),
		Stdin: stdin,
		Env:   proj.env,
		Dir:   proj.dir,
	}, s.budgets.Runs)}
	m.factor = s.budgets.Calibrate(t, s.noop, s.calEnv, perfbudget.RecalibrationRuns)
	s.mu.Lock()
	s.results[key] = m
	s.mu.Unlock()
	return m
}

func payloadNamed(t *testing.T, name string) perfbudget.Payload {
	t.Helper()
	p, ok := perfbudget.PayloadNamed(name)
	if !ok {
		t.Fatalf("no payload %q in the corpus", name)
	}
	return p
}
