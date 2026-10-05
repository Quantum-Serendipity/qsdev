package instance

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// profileTree builds a root with one runnable command per runtime profile,
// named after it, plus an init command marked MarkRootHere, a hook group
// whose subcommand inherits the hook profile, and a stand-in for cobra's
// shell-completion request command.
func profileTree() *cobra.Command {
	run := func(*cobra.Command, []string) error { return nil }
	leaf := func(use string) *cobra.Command { return &cobra.Command{Use: use, RunE: run} }
	root := &cobra.Command{Use: "app"}
	for _, p := range []cmdutil.Profile{
		cmdutil.ProfileInteractive, cmdutil.ProfileGlobal, cmdutil.ProfileUnlogged,
		cmdutil.ProfileAutomatedHook, cmdutil.ProfileMCPServer,
	} {
		root.AddCommand(cmdutil.MarkProfile(leaf(string(p)), p))
	}
	sandbox := &cobra.Command{Use: "sandbox"}
	sandbox.AddCommand(cmdutil.MarkProfile(leaf("exec"), cmdutil.ProfileAutomatedHook), leaf("status"))
	root.AddCommand(sandbox, cmdutil.MarkRootHere(leaf("init")), leaf(cobra.ShellCompRequestCmd))
	return root
}

// isolateLogging points the global log tier at a fresh directory, turns
// logging on and restores the default logger afterwards; it returns the
// global log directory.
func isolateLogging(t *testing.T) string {
	t.Helper()
	b := branding.Get()
	global := t.TempDir()
	t.Setenv(b.EnvLogDirVar, global)
	t.Setenv(b.EnvLogVar, "")
	t.Setenv(b.EnvNoUpdate, "1")
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	return global
}

// chdirProject creates a project (config file at its root) with a sub
// directory, changes into the sub directory and returns the project root.
func chdirProject(t *testing.T) string {
	t.Helper()
	proj := testutil.MarkerFreeTempDir(t)
	if err := os.WriteFile(filepath.Join(proj, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(proj, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	return proj
}

// TestInitLogging_ProfileSelectsTier pins where each profile's session log
// goes when run inside a project: interactive to the project tier, hooks to
// its automated sub-tier, global commands to the global tier, and unlogged
// commands and MCP servers (which open their own session) nowhere.
func TestInitLogging_ProfileSelectsTier(t *testing.T) {
	global := isolateLogging(t)
	proj := chdirProject(t)
	projectLogs := logging.ProjectLogDir(proj)

	tests := []struct {
		args    []string
		wantDir string // "" for no session
	}{
		{[]string{"interactive"}, projectLogs},
		{[]string{"sandbox", "status"}, projectLogs},
		{[]string{"automated-hook"}, filepath.Join(projectLogs, logging.AutomatedLogSubdir)},
		{[]string{"sandbox", "exec", "--", "tool", "--help"}, filepath.Join(projectLogs, logging.AutomatedLogSubdir)},
		{[]string{"global"}, global},
		{[]string{"unlogged"}, ""},
		{[]string{"mcp-server"}, ""},
		{[]string{cobra.ShellCompRequestCmd, "lo"}, ""},
		{nil, ""}, // the bare root
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			rt := &Runtime{}
			rt.initCommand(profileTree(), tt.args, io.Discard, false)
			t.Cleanup(rt.Finish)
			if tt.wantDir == "" {
				if rt.logSession != nil {
					t.Errorf("opened a session in %s, want none", rt.logSession.LogDir)
				}
				return
			}
			if rt.logSession == nil {
				t.Fatalf("opened no session, want one in %s", tt.wantDir)
			}
			if rt.logSession.LogDir != tt.wantDir {
				t.Errorf("session log dir = %s, want %s", rt.logSession.LogDir, tt.wantDir)
			}
			if want := "app " + strings.Join(tt.args, " "); !strings.HasPrefix(want, rt.logSession.Command) {
				t.Errorf("session command = %q, want the executing command's path", rt.logSession.Command)
			}
		})
	}

	// Outside a project the interactive and hook tiers fall back to the
	// global tier.
	t.Run("outside a project", func(t *testing.T) {
		t.Chdir(testutil.MarkerFreeTempDir(t))
		for args, want := range map[string]string{
			"interactive":    global,
			"automated-hook": filepath.Join(global, logging.AutomatedLogSubdir),
		} {
			rt := &Runtime{}
			rt.initCommand(profileTree(), []string{args}, io.Discard, false)
			if rt.logSession == nil || rt.logSession.LogDir != want {
				t.Errorf("%s: session = %+v, want one in %s", args, rt.logSession, want)
			}
			rt.Finish()
		}
	})
}

// TestInitCommand_StoresContext pins that the initializer resolves the
// executing command's project once, with that command's root mode, and
// stores it where cmdutil.Project finds it.
func TestInitCommand_StoresContext(t *testing.T) {
	isolateLogging(t)
	proj := chdirProject(t)
	sub := filepath.Join(proj, "sub")

	for _, tt := range []struct {
		args      []string
		wantRoot  string
		wantFound bool
	}{
		{[]string{"interactive"}, proj, true},
		{[]string{"init", "--yes"}, sub, false},
	} {
		t.Run(tt.args[0], func(t *testing.T) {
			root := profileTree()
			rt := &Runtime{}
			rt.initCommand(root, tt.args, io.Discard, false)
			t.Cleanup(rt.Finish)
			cmd, _, err := root.Find(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			pc, ok := projectctx.FromContext(cmd.Context())
			if !ok {
				t.Fatalf("no project stored on %q", cmd.CommandPath())
			}
			if pc.Root != tt.wantRoot || pc.Found != tt.wantFound {
				t.Errorf("stored project = {Root: %s, Found: %v}, want {Root: %s, Found: %v}", pc.Root, pc.Found, tt.wantRoot, tt.wantFound)
			}
			got, err := cmdutil.Project(cmd)
			if err != nil || got.Root != tt.wantRoot {
				t.Errorf("cmdutil.Project = %q, %v; want the stored %q", got.Root, err, tt.wantRoot)
			}
		})
	}

	// Nothing is stored on the bare root: every subcommand inherits its
	// context, so a stored project would outlive the execution.
	t.Run("bare root", func(t *testing.T) {
		root := profileTree()
		rt := &Runtime{}
		rt.initCommand(root, []string{"-test.v"}, io.Discard, false)
		t.Cleanup(rt.Finish)
		if ctx := root.Context(); ctx != nil {
			if _, ok := projectctx.FromContext(ctx); ok {
				t.Error("a project was stored on the root command")
			}
		}
	})
}

// TestUpdateCheckSkippedForHookAndMCPProfiles pins that a hook or MCP server
// process starts no update check, even with a terminal on stderr and a fresh
// cache naming a newer release, while an interactive command does.
func TestUpdateCheckSkippedForHookAndMCPProfiles(t *testing.T) {
	isolateLogging(t)
	t.Setenv(branding.Get().EnvNoUpdate, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeUpdateCache(t, home, time.Now().UTC())
	t.Chdir(testutil.MarkerFreeTempDir(t))

	for _, tt := range []struct {
		profile   cmdutil.Profile
		wantCheck bool
	}{
		{cmdutil.ProfileAutomatedHook, false},
		{cmdutil.ProfileMCPServer, false},
		{cmdutil.ProfileInteractive, true},
		{cmdutil.ProfileGlobal, true},
	} {
		t.Run(string(tt.profile), func(t *testing.T) {
			rt := &Runtime{}
			rt.initCommand(profileTree(), []string{string(tt.profile)}, io.Discard, true)
			if got := rt.updateCh != nil; got != tt.wantCheck {
				t.Errorf("update check started = %v, want %v", got, tt.wantCheck)
			}
			if rt.updateCh != nil {
				select {
				case <-rt.updateCh:
				case <-time.After(10 * time.Second):
					t.Fatal("timed out waiting for the cached notice")
				}
			}
			if rt.logSession != nil {
				rt.logSession.Close()
			}
		})
	}
}

// TestIgnoredMarkerNoticeInteractiveOnly pins that when no project was found
// but untrusted markers were skipped, an interactive command says so in one
// stderr line (so a CI uid mismatch is not silent), and no other profile
// prints anything.
func TestIgnoredMarkerNoticeInteractiveOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("marker trust is ACL-based on Windows and out of scope")
	}
	isolateLogging(t)
	b := branding.Get()
	base := testutil.MarkerFreeTempDir(t)
	shared := filepath.Join(base, "shared")
	victim := filepath.Join(shared, "victim")
	if err := os.MkdirAll(filepath.Join(shared, b.StateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shared, 0o1777); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(shared, 0o755) })
	t.Chdir(victim)

	for _, tt := range []struct {
		args       []string
		wantNotice bool
	}{
		{[]string{"interactive"}, true},
		{[]string{"global"}, false},
		{[]string{"unlogged"}, false},
		{[]string{"automated-hook"}, false},
		{[]string{"mcp-server"}, false},
	} {
		t.Run(tt.args[0], func(t *testing.T) {
			var stderr bytes.Buffer
			rt := &Runtime{}
			rt.initCommand(profileTree(), tt.args, &stderr, false)
			t.Cleanup(rt.Finish)
			out := stderr.String()
			if !tt.wantNotice {
				if out != "" {
					t.Errorf("printed %q, want nothing", out)
				}
				return
			}
			if strings.Count(out, "\n") != 1 || !strings.Contains(out, filepath.Join(shared, b.StateDir)) {
				t.Errorf("notice = %q, want one line naming the ignored marker", out)
			}
		})
	}

	// A trusted project found below is no reason for a note.
	t.Run("project found", func(t *testing.T) {
		chdirProject(t)
		var stderr bytes.Buffer
		rt := &Runtime{}
		rt.initCommand(profileTree(), []string{"interactive"}, &stderr, false)
		t.Cleanup(rt.Finish)
		if stderr.Len() != 0 {
			t.Errorf("printed %q inside a project, want nothing", stderr.String())
		}
	})
}

// TestLegacyLogsMigratedForHumanProfilesOnly pins the one-time move of the
// legacy ~/.<app>/logs to the per-user state directory: a command a person
// runs (interactive, global or unlogged) moves it before opening its session
// log, a hook or MCP server process never touches it, and a log-dir override
// (the user chose their own log location) leaves it alone.
func TestLegacyLogsMigratedForHumanProfilesOnly(t *testing.T) {
	b := branding.Get()
	tests := []struct {
		profile  cmdutil.Profile
		override bool
		wantMove bool
	}{
		{profile: cmdutil.ProfileInteractive, wantMove: true},
		{profile: cmdutil.ProfileGlobal, wantMove: true},
		{profile: cmdutil.ProfileUnlogged, wantMove: true},
		{profile: cmdutil.ProfileAutomatedHook},
		{profile: cmdutil.ProfileMCPServer},
		{profile: cmdutil.ProfileInteractive, override: true},
	}
	for _, tt := range tests {
		name := string(tt.profile)
		if tt.override {
			name += "-with-override"
		}
		t.Run(name, func(t *testing.T) {
			isolateLogging(t)
			base := t.TempDir()
			home := filepath.Join(base, "home")
			state := filepath.Join(base, "state")
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_STATE_HOME", state)
			if !tt.override {
				t.Setenv(b.EnvLogDirVar, "")
			}
			legacyLogs := filepath.Join(home, "."+b.AppName, "logs")
			if err := os.MkdirAll(legacyLogs, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(legacyLogs, "old.jsonl"), []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Chdir(testutil.MarkerFreeTempDir(t))

			rt := &Runtime{}
			rt.initCommand(profileTree(), []string{string(tt.profile)}, io.Discard, false)
			t.Cleanup(rt.Finish)

			_, legacyErr := os.Stat(filepath.Join(legacyLogs, "old.jsonl"))
			_, movedErr := os.Stat(filepath.Join(state, b.AppName, "logs", "old.jsonl"))
			if moved := legacyErr != nil && movedErr == nil; moved != tt.wantMove {
				t.Errorf("legacy logs moved = %v, want %v (legacy stat: %v, state stat: %v)", moved, tt.wantMove, legacyErr, movedErr)
			}
		})
	}
}

// TestLegacyLogsMigratedAfterHookRun covers the usual order under Claude Code:
// after an upgrade a hook (automated-hook profile, which never migrates) runs
// first and opens its session in the new global tier, then a person runs a
// command. The legacy logs must still move then.
func TestLegacyLogsMigratedAfterHookRun(t *testing.T) {
	b := branding.Get()
	isolateLogging(t)
	base := t.TempDir()
	home := filepath.Join(base, "home")
	state := filepath.Join(base, "state")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv(b.EnvLogDirVar, "")
	legacyLogs := filepath.Join(home, "."+b.AppName, "logs")
	if err := os.MkdirAll(legacyLogs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyLogs, "old.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(testutil.MarkerFreeTempDir(t))
	stateLogs := filepath.Join(state, b.AppName, "logs")

	hook := &Runtime{}
	hook.initCommand(profileTree(), []string{string(cmdutil.ProfileAutomatedHook)}, io.Discard, false)
	hook.Finish()
	if _, err := os.Stat(stateLogs); err != nil {
		t.Fatalf("precondition: the hook session did not create %s: %v", stateLogs, err)
	}

	rt := &Runtime{}
	rt.initCommand(profileTree(), []string{string(cmdutil.ProfileInteractive)}, io.Discard, false)
	t.Cleanup(rt.Finish)
	if _, err := os.Stat(filepath.Join(stateLogs, "old.jsonl")); err != nil {
		t.Errorf("legacy log not moved after a hook run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyLogs, "old.jsonl")); err == nil {
		t.Errorf("legacy log still at %s", legacyLogs)
	}
}
