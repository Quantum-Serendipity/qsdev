package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Guardrail invariant tests (XS-WS2 B1, B2, B5). They drive the real command
// tree in throwaway projects: whatever an (agent- or human-) edited answers
// file says, every command that writes .claude/settings.json registers the
// self-protection hook, and the package-guard hook unless the committed
// .qsdev.yaml records the opt-out (`disable <safety-block tool> --force`).

// cliHelperEnv makes the test binary run as qsdev itself (see TestMain), so
// each command runs in a fresh process exactly as the shipped binary does:
// the command tree keeps flag values between executions in one process.
const cliHelperEnv = "QSDEV_GUARDRAIL_CLI_HELPER"

// guardrailEnv returns the environment for qsdev processes: the caller's,
// minus anything naming a home, temp, user config, Claude Code session or
// qsdev setting, with HOME, TMPDIR and the user config directories (XDG and
// the Windows APPDATA/LOCALAPPDATA) pointed at
// fresh temp dirs and system setup skipped, so the commands see no state
// from the machine running the tests. PATH starts with guardrailBinDir, so
// qsdev resolves to the build under test and no container runtime probe
// reaches the host's. The commands run as a human at a terminal
// (humanHelperEnv), as the documented commands are; agentEnv turns that off.
func guardrailEnv(t *testing.T) []string {
	t.Helper()
	b := branding.Get()
	var env []string
	path := guardrailBinDir(t)
	for _, kv := range os.Environ() {
		name, value, _ := strings.Cut(kv, "=")
		switch {
		case name == "GORACE": // replaced by raceOptions below
			continue
		case strings.EqualFold(name, "PATH"): // Windows spells it Path
			if value != "" {
				path += string(os.PathListSeparator) + value
			}
			continue
		case name == "HOME", name == "TMPDIR", name == "TMP", name == "TEMP", name == "USERPROFILE",
			strings.EqualFold(name, "APPDATA"), strings.EqualFold(name, "LOCALAPPDATA"),
			strings.HasPrefix(name, "XDG_"), strings.HasPrefix(name, "CLAUDE"),
			strings.HasPrefix(name, b.EnvPrefix):
			continue
		}
		env = append(env, kv)
	}
	home, tmp := t.TempDir(), t.TempDir()
	return append(env,
		"PATH="+path,
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		// os.UserConfigDir and the installed-binary location on Windows.
		"APPDATA="+filepath.Join(home, "AppData", "Roaming"),
		"LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"),
		"TMPDIR="+tmp, "TMP="+tmp, "TEMP="+tmp,
		b.EnvPrefix+"SKIP_SETUP=1",
		b.EnvNoUpdate+"=1",
		cliHelperEnv+"=1",
		humanHelperEnv+"=1",
		raceOptions(),
	)
}

// raceOptions returns the caller's GORACE setting with the race runtime's
// exit delay switched off. A -race build sleeps atexit_sleep_ms (default one
// second) before every exit, and these tests start hundreds of qsdev
// processes; reports and exit codes are unaffected.
func raceOptions() string {
	return "GORACE=" + strings.TrimSpace(os.Getenv("GORACE")+" atexit_sleep_ms=0")
}

// agentEnv returns env as an AI agent's tool call sees it: inside a Claude
// Code session (CLAUDECODE=1) with no terminal.
func agentEnv(env []string) []string {
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		return strings.HasPrefix(kv, humanHelperEnv+"=")
	})
	return append(out, "CLAUDECODE=1")
}

// newGuardrailProject returns a fresh Go project with its own git repository,
// so root resolution stops at it.
func newGuardrailProject(t *testing.T, env []string) string {
	t.Helper()
	dir := t.TempDir()
	git := exec.Command("git", "init", "-q", dir)
	git.Env = env
	if out, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/guardrail\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runQsdev runs qsdev with args in dir and returns its combined output and
// exit code.
func runQsdev(t *testing.T, env []string, dir string, stdin io.Reader, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &exitErr):
		return string(out), exitErr.ExitCode()
	}
	t.Fatalf("running qsdev %s: %v", strings.Join(args, " "), err)
	return "", -1
}

// mustQsdev runs args and fails the test when the command fails.
func mustQsdev(t *testing.T, env []string, dir string, args ...string) {
	t.Helper()
	if out, code := runQsdev(t, env, dir, nil, args...); code != 0 {
		t.Fatalf("qsdev %s: exit %d\n%s", strings.Join(args, " "), code, out)
	}
}

// safetyBlockTool returns the catalog tool behind the package-guard hook: the
// one tool that exclusively owns a Claude Code hook script.
func safetyBlockTool(t *testing.T) *toolreg.Tool {
	t.Helper()
	var found []*toolreg.Tool
	for _, tool := range toolreg.DefaultRegistry().All() {
		if slices.ContainsFunc(tool.ExclusiveFiles(), isHookScript) {
			found = append(found, tool)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one catalog tool owning a .claude/hooks script, got %d", len(found))
	}
	return found[0]
}

func isHookScript(f toolreg.FileOwnership) bool {
	return strings.HasPrefix(f.Path, ".claude/hooks/")
}

// packageGuardScript returns the project-relative path of the package-guard
// script, as the catalog records it.
func packageGuardScript(t *testing.T) string {
	t.Helper()
	i := slices.IndexFunc(safetyBlockTool(t).ExclusiveFiles(), isHookScript)
	return safetyBlockTool(t).ExclusiveFiles()[i].Path
}

// toggleTool returns the first catalog tool, by name, that is neither a
// security tool nor on by default and has no prerequisites or conflicts: a
// tool enable and disable can switch freely in any project.
func toggleTool(t *testing.T) string {
	t.Helper()
	tools := toolreg.DefaultRegistry().All()
	slices.SortFunc(tools, func(a, b *toolreg.Tool) int { return strings.Compare(a.Name, b.Name) })
	for _, tool := range tools {
		if tool.Category != toolreg.CategorySecurity && tool.Default == toolreg.OptIn &&
			len(tool.Prerequisites) == 0 && len(tool.Conflicts) == 0 {
			return tool.Name
		}
	}
	t.Fatal("catalog has no opt-in non-security tool")
	return ""
}

// hookSubset is one hostile hook selection written into the answers files.
type hookSubset struct {
	name  string
	hooks types.HookChoices
	// forgeToolOff also records the safety-block tool off, as `disable
	// --force` does, without the committed config recording it.
	forgeToolOff bool
}

// hostileHookSubsets returns the adversarial selections: every hook off,
// every hook on, every hook on but one, and every hook off with a safety
// block opt-out the committed config does not record, with and without the
// safety-block tool also recorded off.
func hostileHookSubsets(t *testing.T) []hookSubset {
	t.Helper()
	names := types.HookChoiceNames()
	choices := func(skip string) types.HookChoices {
		var h types.HookChoices
		for _, name := range names {
			if name == skip {
				continue
			}
			if err := h.EnableHook(name); err != nil {
				t.Fatalf("EnableHook(%q): %v", name, err)
			}
		}
		return h
	}
	subsets := []hookSubset{
		{name: "all-false"},
		{name: "all-true", hooks: choices("")},
		{name: "forged-opt-out", hooks: types.HookChoices{SafetyBlockOptOut: true}},
		{name: "forged-tool-opt-out", hooks: types.HookChoices{SafetyBlockOptOut: true}, forgeToolOff: true},
	}
	for _, name := range names {
		subsets = append(subsets, hookSubset{name: name + "-off", hooks: choices(name)})
	}
	return subsets
}

// answersFiles returns the project-relative paths of every answers file a
// command may read: the primary file, the devenv mirror and the legacy
// Claude Code copy.
func answersFiles() []string {
	return []string{
		path.Join(answers.PrimaryDir(), answers.PrimaryFilename()),
		answers.DevenvCopyFile(),
		answers.LegacyClaudeCopyFile(),
	}
}

// seedHostileAnswers writes the subset's hook selection into every answers
// file in dir, creating the primary file when the project has none. An
// existing file keeps its recorded opt-out unless the subset forges one.
func seedHostileAnswers(t *testing.T, dir string, s hookSubset) {
	t.Helper()
	for i, rel := range answersFiles() {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err != nil && i > 0 {
			continue
		}
		relDir, file := path.Split(rel)
		// A missing primary file is seeded from scratch.
		a, err := answers.LoadFromDir(dir, relDir, file, "init")
		if err != nil {
			a = types.WizardAnswers{}
		}
		hooks := s.hooks
		hooks.SafetyBlockOptOut = s.hooks.SafetyBlockOptOut || a.Hooks.SafetyBlockOptOut
		a.Hooks = hooks
		if s.forgeToolOff {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools[safetyBlockTool(t).Name] = false
		}
		if err := answers.SaveToDir(dir, relDir, file, a); err != nil {
			t.Fatalf("seeding %s: %v", rel, err)
		}
	}
}

// preToolUseCommands returns every PreToolUse hook command in dir's
// .claude/settings.json.
func preToolUseCommands(t *testing.T, dir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("reading settings.json: %v", err)
	}
	var settings struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	var cmds []string
	for _, m := range settings.Hooks["PreToolUse"] {
		for _, h := range m.Hooks {
			cmds = append(cmds, h.Command)
		}
	}
	return cmds
}

func anyContains(cmds []string, sub string) bool {
	return slices.ContainsFunc(cmds, func(c string) bool { return strings.Contains(c, sub) })
}

// assertGuardrails checks dir's settings.json registers selfprotect, and
// package-guard exactly when no opt-out is committed.
func assertGuardrails(t *testing.T, dir string, optedOut bool) {
	t.Helper()
	cmds := preToolUseCommands(t, dir)
	if self := branding.Get().AppName + " selfprotect"; !anyContains(cmds, self) {
		t.Errorf("PreToolUse has no %q hook:\n%s", self, strings.Join(cmds, "\n"))
	}
	script := path.Base(packageGuardScript(t))
	if got := anyContains(cmds, script); got == optedOut {
		t.Errorf("PreToolUse runs %s = %v, want %v (committed opt-out %v):\n%s", script, got, !optedOut, optedOut, strings.Join(cmds, "\n"))
	}
}

// initialisedProject returns a project after `init --yes`, with the safety
// block opted out through the documented command when optedOut is set.
func initialisedProject(t *testing.T, env []string, optedOut bool) (string, bool) {
	t.Helper()
	return cachedProject(t, "initialised", env, optedOut, prepareInitialisedProject)
}

// prepareInitialisedProject runs the commands behind initialisedProject.
func prepareInitialisedProject(t *testing.T, env []string, optedOut bool) (string, bool) {
	t.Helper()
	dir := newGuardrailProject(t, env)
	mustQsdev(t, env, dir, "init", "--yes")
	if optedOut {
		mustQsdev(t, env, dir, "disable", safetyBlockTool(t).Name, "--force")
	}
	return dir, optedOut
}

// claudeOnlyProject returns a project after `claude init --yes`, which writes
// no committed .qsdev.yaml. The documented opt-out needs one, so when optedOut
// is set it checks that `disable --force` is refused and reports that no
// opt-out is committed.
func claudeOnlyProject(t *testing.T, env []string, optedOut bool) (string, bool) {
	t.Helper()
	return cachedProject(t, "claude-only", env, optedOut, prepareClaudeOnlyProject)
}

// prepareClaudeOnlyProject runs the commands behind claudeOnlyProject.
func prepareClaudeOnlyProject(t *testing.T, env []string, optedOut bool) (string, bool) {
	t.Helper()
	dir := newGuardrailProject(t, env)
	mustQsdev(t, env, dir, "claude", "init", "--yes")
	if _, err := os.Stat(filepath.Join(dir, branding.Get().ConfigFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("claude init wrote %s (err %v); this preparer needs a project without one", branding.Get().ConfigFile, err)
	}
	if optedOut {
		if out, code := runQsdev(t, env, dir, nil, "disable", safetyBlockTool(t).Name, "--force"); code == 0 {
			t.Fatalf("disable --force succeeded without a committed config, want it refused:\n%s", out)
		}
	}
	return dir, false
}

// commandPath is one documented way to (re)generate .claude/settings.json.
type commandPath struct {
	name string
	// prepare readies a project for the command, opting out of the safety
	// block when optedOut is set, and returns the directory the command runs
	// in and whether an opt-out is now committed.
	prepare func(t *testing.T, env []string, optedOut bool) (string, bool)
	args    []string
	// initialisedArgs, when set, replaces args in an initialised project.
	initialisedArgs []string
	// maxExit is the highest exit code that still means success.
	maxExit int
}

// argsFor returns the command line for a project prepared with optedOut.
func (cp commandPath) argsFor(optedOut bool) []string {
	if optedOut && cp.initialisedArgs != nil {
		return cp.initialisedArgs
	}
	return cp.args
}

// freshOrInitialised prepares a fresh project, or an initialised one when an
// opt-out must be committed first.
func freshOrInitialised(t *testing.T, env []string, optedOut bool) (string, bool) {
	t.Helper()
	if optedOut {
		return initialisedProject(t, env, true)
	}
	return newGuardrailProject(t, env), false
}

func guardrailCommandPaths(t *testing.T) []commandPath {
	t.Helper()
	tool := toggleTool(t)
	return []commandPath{
		{name: "init -y", prepare: freshOrInitialised, args: []string{"init", "--yes"}},
		{
			name:    "claude init --yes",
			prepare: freshOrInitialised,
			args:    []string{"claude", "init", "--yes"},
			// Over existing Claude Code files claude init needs --force.
			initialisedArgs: []string{"claude", "init", "--yes", "--force"},
		},
		{name: "claude update", prepare: initialisedProject, args: []string{"claude", "update"}},
		{name: "claude update (claude-only)", prepare: claudeOnlyProject, args: []string{"claude", "update"}},
		{name: "claude add-hook audit-log", prepare: initialisedProject, args: []string{"claude", "add-hook", "audit-log"}},
		{name: "claude add-hook audit-log (claude-only)", prepare: claudeOnlyProject, args: []string{"claude", "add-hook", "audit-log"}},
		{name: "init --update -y", prepare: initialisedProject, args: []string{"init", "--update", "--yes"}},
		{name: "enable " + tool, prepare: initialisedProject, args: []string{"enable", tool}},
		{
			name: "disable " + tool,
			prepare: func(t *testing.T, env []string, optedOut bool) (string, bool) {
				t.Helper()
				dir, committed := initialisedProject(t, env, optedOut)
				mustQsdev(t, env, dir, "enable", tool)
				return dir, committed
			},
			args: []string{"disable", tool},
		},
		{
			// Repair must regenerate settings.json, so it starts deleted.
			name: "repair --force",
			prepare: func(t *testing.T, env []string, optedOut bool) (string, bool) {
				t.Helper()
				dir, committed := initialisedProject(t, env, optedOut)
				if err := os.Remove(filepath.Join(dir, ".claude", "settings.json")); err != nil {
					t.Fatal(err)
				}
				return dir, committed
			},
			args: []string{"repair", "--force"},
			// Repair exits 1 when it leaves a finding to the user (here the
			// missing go.sum and the uninstalled git hooks); 2 means it failed.
			maxExit: 1,
		},
		{
			// A teammate's clone carries only the committed .qsdev.yaml.
			name: "init --mode join",
			prepare: func(t *testing.T, env []string, optedOut bool) (string, bool) {
				t.Helper()
				origin, committed := initialisedProject(t, env, optedOut)
				clone := newGuardrailProject(t, env)
				cfg := branding.Get().ConfigFile
				data, err := os.ReadFile(filepath.Join(origin, cfg))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(clone, cfg), data, 0o644); err != nil {
					t.Fatal(err)
				}
				return clone, committed
			},
			args: []string{"init", "--mode", "join", "--yes"},
		},
	}
}

// TestGuardrailInvariant_EveryCommandPath is B1 at the command level: no
// documented command, fed a hostile answers file, writes a settings.json
// without selfprotect, or without package-guard unless the opt-out is
// committed. The generator-level matrix over every hook subset lives in
// addons/claudecode; this proves no command bypasses the generator.
func TestGuardrailInvariant_EveryCommandPath(t *testing.T) {
	t.Parallel()
	subsets := hostileHookSubsets(t)
	for _, cp := range guardrailCommandPaths(t) {
		t.Run(cp.name, func(t *testing.T) {
			t.Parallel()
			for _, optedOut := range []bool{false, true} {
				for _, s := range subsets {
					name := s.name
					if optedOut {
						name = "opted-out/" + name
					}
					t.Run(name, func(t *testing.T) {
						t.Parallel()
						env := guardrailEnv(t)
						// Every subset of a path starts from the same project.
						dir, committed := cachedProject(t, "path "+cp.name, env, optedOut, cp.prepare)
						seedHostileAnswers(t, dir, s)
						args := cp.argsFor(committed)
						if out, code := runQsdev(t, env, dir, nil, args...); code > cp.maxExit {
							t.Fatalf("qsdev %s: exit %d\n%s", strings.Join(args, " "), code, out)
						}
						assertGuardrails(t, dir, committed)
					})
				}
			}
		})
	}
}

// TestGuardrailInvariant_HumanSedSelfProtection is B2: a human switching
// self_protection off in the answers file does not drop the hook on the next
// `claude update`, and the hook itself denies an agent making that edit.
func TestGuardrailInvariant_HumanSedSelfProtection(t *testing.T) {
	t.Parallel()
	env := guardrailEnv(t)
	dir := newGuardrailProject(t, env)
	mustQsdev(t, env, dir, "claude", "init", "--yes")
	assertGuardrails(t, dir, false)

	answersPath := answers.PrimaryPath(dir)
	data, err := os.ReadFile(answersPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "self_protection: true", "self_protection: false", 1)
	if edited == string(data) {
		t.Fatalf("answers file records no self_protection: true:\n%s", data)
	}
	if err := os.WriteFile(answersPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	mustQsdev(t, env, dir, "claude", "update")
	assertGuardrails(t, dir, false)

	// The agent-side edit is denied through the real hook IO (B3 wiring;
	// the rules-level verdict table is in internal/selfprotect/rules).
	rel := path.Join(answers.PrimaryDir(), answers.PrimaryFilename())
	payload, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"cwd":             dir,
		"tool_name":       "Bash",
		"tool_input":      map[string]string{"command": "sed -i s/true/false/ " + rel},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, code := runQsdev(t, env, dir, bytes.NewReader(payload), "selfprotect")
	if code != 2 {
		t.Fatalf("selfprotect on %q: exit %d, want 2\n%s", rel, code, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("selfprotect denied without a reason on stderr")
	}
}

// TestGuardrailInvariant_ClaudeHooksAdditive is B5: --claude-hooks adds to
// the security hooks rather than replacing them, and check fails once
// package-guard's script or registration is removed.
func TestGuardrailInvariant_ClaudeHooksAdditive(t *testing.T) {
	t.Parallel()
	script := packageGuardScript(t)
	initAdditive := func(t *testing.T) (env []string, dir string) {
		t.Helper()
		env = guardrailEnv(t)
		dir = newGuardrailProject(t, env)
		mustQsdev(t, env, dir, "init", "--yes", "--claude-hooks", "audit-log,credential-scan")
		return env, dir
	}

	t.Run("generated and checked", func(t *testing.T) {
		t.Parallel()
		env, dir := initAdditive(t)
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(script))); err != nil {
			t.Fatalf("%s not generated: %v", script, err)
		}
		assertGuardrails(t, dir, false)
		mustQsdev(t, env, dir, "check")
	})

	t.Run("script deleted", func(t *testing.T) {
		t.Parallel()
		env, dir := initAdditive(t)
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(script))); err != nil {
			t.Fatal(err)
		}
		assertCheckFails(t, env, dir, path.Base(script))
	})

	t.Run("registration removed", func(t *testing.T) {
		t.Parallel()
		env, dir := initAdditive(t)
		settingsPath := filepath.Join(dir, ".claude", "settings.json")
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(settingsPath, withoutHookCommand(t, data, path.Base(script)), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCheckFails(t, env, dir, "claude_hook_missing")
	})

	t.Run("unimplemented preset rejected", func(t *testing.T) {
		t.Parallel()
		env := guardrailEnv(t)
		dir := newGuardrailProject(t, env)
		if out, code := runQsdev(t, env, dir, nil, "init", "--yes", "--claude-hooks", "auto-format,audit-log"); code == 0 {
			t.Fatalf("init --claude-hooks auto-format,audit-log succeeded, want an error:\n%s", out)
		}
	})
}

// assertCheckFails runs check in dir and requires it to exit non-zero with a
// FAIL line naming want.
func assertCheckFails(t *testing.T, env []string, dir, want string) {
	t.Helper()
	out, code := runQsdev(t, env, dir, nil, "check")
	if code == 0 {
		t.Fatalf("check passed, want a failure naming %q:\n%s", want, out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "FAIL") && strings.Contains(line, want) {
			return
		}
	}
	t.Errorf("check output has no FAIL line naming %q:\n%s", want, out)
}

// withoutHookCommand returns settings.json data with every hook whose
// command mentions sub removed, leaving the rest of the file untouched.
func withoutHookCommand(t *testing.T, data []byte, sub string) []byte {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	hooks, _ := settings["hooks"].(map[string]any)
	removed := false
	for event, raw := range hooks {
		matchers, _ := raw.([]any)
		var kept []any
		for _, rm := range matchers {
			m, _ := rm.(map[string]any)
			entries, _ := m["hooks"].([]any)
			var keptEntries []any
			for _, re := range entries {
				e, _ := re.(map[string]any)
				if cmd, _ := e["command"].(string); strings.Contains(cmd, sub) {
					removed = true
					continue
				}
				keptEntries = append(keptEntries, re)
			}
			if len(keptEntries) > 0 {
				m["hooks"] = keptEntries
				kept = append(kept, m)
			}
		}
		hooks[event] = kept
	}
	if !removed {
		t.Fatalf("settings.json has no hook running %q", sub)
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return out
}
