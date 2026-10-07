package claudecode

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// recordingRunner records every package-manager command and fails it, so a
// test can assert that no command ran.
type recordingRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil, errors.New("recordingRunner: no command may run")
}

// writeProjectConfig writes content as dir's committed project config; ""
// writes none.
func writeProjectConfig(t *testing.T, dir, content string) {
	t.Helper()
	if content == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestNewLifecycle_MinReleaseAgeFromSecurityLevel checks that `mcp install`
// and `mcp update` gate releases at the project's compliance tier, and that a
// project without a config keeps today's baseline (0, the manager floors).
func TestNewLifecycle_MinReleaseAgeFromSecurityLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config string
		want   time.Duration
	}{
		{"strict", "version: 1\nsecurity:\n  level: strict\n", catalog.EffectiveAgeGate("strict")},
		{"baseline", "version: 1\nsecurity:\n  level: baseline\n", catalog.EffectiveAgeGate("baseline")},
		{"client raises the floor", "version: 1\nsecurity:\n  level: baseline\nclient:\n  security_level: strict\n", catalog.EffectiveAgeGate("strict")},
		{"no project config", "", 0},
	}
	if catalog.EffectiveAgeGate("strict") <= catalog.EffectiveAgeGate("baseline") {
		t.Fatal("the strict age gate must exceed the baseline one for this test to discriminate")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeProjectConfig(t, dir, tt.config)
			runner := &recordingRunner{}
			lc, err := newLifecycle(dir, runner)
			if err != nil {
				t.Fatalf("newLifecycle: %v", err)
			}
			if lc.MinReleaseAge != tt.want {
				t.Errorf("MinReleaseAge = %v, want %v", lc.MinReleaseAge, tt.want)
			}
			if lc.CmdRunner != runner {
				t.Error("newLifecycle did not wire the given command runner")
			}
		})
	}
}

// TestMCPInstall_MalformedConfigFailsClosed checks that a project config that
// exists but cannot be loaded stops `mcp install` and `mcp update` before any
// package-manager command runs, instead of installing at the baseline gate.
func TestMCPInstall_MalformedConfigFailsClosed(t *testing.T) {
	t.Parallel()
	name := packageManagedServer(t)
	tests := []struct {
		cmd  string
		new  func(mcpregistry.CommandRunner) *cobra.Command
		args []string
	}{
		{"install", newMCPInstallCmd, []string{name}},
		{"update", newMCPUpdateCmd, []string{name}},
		{"update --all", newMCPUpdateCmd, []string{"--all"}},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeProjectConfig(t, dir, "version: [unterminated\n")
			runner := &recordingRunner{}
			cmd := tt.new(runner)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tt.args)
			cmd.SetContext(projectctx.WithContext(context.Background(), projectctx.Context{Start: dir, Root: dir}))
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), branding.Get().ConfigFile) {
				t.Fatalf("err = %v, want an error naming the malformed %s", err, branding.Get().ConfigFile)
			}
			if len(runner.calls) != 0 {
				t.Errorf("package-manager commands ran despite the malformed config: %v", runner.calls)
			}
		})
	}
}

// failingRunner fails every package-manager command.
type failingRunner struct{}

func (failingRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return []byte("registry unreachable"), errors.New("exit status 1")
}

// packageManagedServer returns a registry server installed by a package
// manager, so its lifecycle operations run a command.
func packageManagedServer(t *testing.T) string {
	t.Helper()
	for _, def := range mcpregistry.DefaultRegistry().All() {
		if def.PackageName != "" && (def.InstallMethod == mcpregistry.InstallNpmGlobal || def.InstallMethod == mcpregistry.InstallUvTool) {
			return def.Name
		}
	}
	t.Skip("no package-managed MCP server in the registry")
	return ""
}

func failingLifecycle(recorded ...string) *mcpregistry.McpLifecycle {
	st := &types.GeneratedState{McpServers: map[string]types.McpServerState{}}
	for _, name := range recorded {
		st.McpServers[name] = types.McpServerState{InstalledVersion: "1.0.0"}
	}
	return &mcpregistry.McpLifecycle{
		CmdRunner:   failingRunner{},
		StateLoader: func() (*types.GeneratedState, error) { return st, nil },
		StateSaver:  func(s *types.GeneratedState) error { st = s; return nil },
	}
}

// TestMCPLifecycle_FailuresExitNonZero guards F096: a failed install, update
// or remove must return an error so the command exits non-zero.
func TestMCPLifecycle_FailuresExitNonZero(t *testing.T) {
	t.Parallel()
	name := packageManagedServer(t)
	ctx := context.Background()

	tests := []struct {
		op  string
		run func(*bytes.Buffer) error
	}{
		{"install", func(w *bytes.Buffer) error { return runMCPInstall(ctx, w, failingLifecycle(), name) }},
		{"update", func(w *bytes.Buffer) error { return runMCPUpdate(ctx, w, failingLifecycle(name), name) }},
		{"update --all", func(w *bytes.Buffer) error { return runMCPUpdateAll(ctx, w, failingLifecycle(name)) }},
		{"remove", func(w *bytes.Buffer) error { return runMCPRemove(ctx, w, failingLifecycle(name), name) }},
	}
	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := tc.run(&out)
			if !errors.Is(err, errMCPOperationFailed) {
				t.Errorf("err = %v, want errMCPOperationFailed", err)
			}
			if !bytes.Contains(out.Bytes(), []byte("Could not")) {
				t.Errorf("expected the failure to be printed, got %q", out.String())
			}
		})
	}
}

// TestStateFilePath_UsesBrandedClaudeState guards F093: MCP lifecycle records
// share the claude addon's branded state file.
func TestStateFilePath_UsesBrandedClaudeState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got, want := stateFilePath(root), filepath.Join(root, statePath()); got != want {
		t.Errorf("stateFilePath = %q, want %q", got, want)
	}
}
