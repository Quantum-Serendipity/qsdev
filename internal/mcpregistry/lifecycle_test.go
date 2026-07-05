package mcpregistry

import (
	"context"
	"errors"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type mockRunner struct {
	commands [][]string
	err      error
	// responder, when set, returns the output and error for a given command,
	// overriding err. It lets a test make the install command succeed while
	// the version-list probe returns crafted output.
	responder func(name string, args []string) ([]byte, error)
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.commands = append(m.commands, append([]string{name}, args...))
	if m.responder != nil {
		return m.responder(name, args)
	}
	return nil, m.err
}

func testStateLoader(state *types.GeneratedState) func() (*types.GeneratedState, error) {
	return func() (*types.GeneratedState, error) { return state, nil }
}

func testStateSaver(state *types.GeneratedState) func(*types.GeneratedState) error {
	return func(s *types.GeneratedState) error { *state = *s; return nil }
}

func newTestLifecycle(runner *mockRunner, state *types.GeneratedState) *McpLifecycle {
	return &McpLifecycle{
		CmdRunner:   runner,
		StateLoader: testStateLoader(state),
		StateSaver:  testStateSaver(state),
	}
}

func TestInstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverName  string
		wantCmd     []string
		wantInstall bool
		wantErr     bool
	}{
		{
			name:        "UvTool",
			serverName:  "man-pages",
			wantCmd:     []string{"uv", "tool", "install", "man-mcp-server"},
			wantInstall: true,
		},
		{
			name:        "NpmGlobal",
			serverName:  "local-docs-devdocs",
			wantCmd:     []string{"npm", "install", "-g", "@madhan-g-p/devdocs-mcp-server"},
			wantInstall: true,
		},
		{
			name:       "Unknown",
			serverName: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := &mockRunner{}
			state := &types.GeneratedState{
				McpServers: make(map[string]types.McpServerState),
			}
			lc := newTestLifecycle(runner, state)

			result, err := lc.Install(context.Background(), tt.serverName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Installed != tt.wantInstall {
				t.Errorf("Installed = %v, want %v", result.Installed, tt.wantInstall)
			}

			if tt.wantCmd != nil {
				// A successful install runs the install command first, then a
				// version-resolution probe, so assert on the first command.
				if len(runner.commands) == 0 {
					t.Fatal("expected at least 1 command, got 0")
				}
				got := runner.commands[0]
				if len(got) != len(tt.wantCmd) {
					t.Fatalf("command = %v, want %v", got, tt.wantCmd)
				}
				for i := range got {
					if got[i] != tt.wantCmd[i] {
						t.Errorf("command[%d] = %q, want %q", i, got[i], tt.wantCmd[i])
					}
				}
			}
		})
	}
}

func TestInstall_NixPackage(t *testing.T) {
	t.Parallel()

	// Register a temporary nix-package server for testing.
	reg := DefaultRegistry()
	testDef := McpServerDefinition{
		Name:          "test-nix-lifecycle",
		DisplayName:   "Test Nix",
		Command:       "test",
		Transport:     TransportStdio,
		Source:        SourceBuiltin,
		InstallMethod: InstallNixPackage,
		PackageName:   "test-pkg",
	}
	_ = reg.Register(testDef)
	t.Cleanup(func() {
		reg.Delete("test-nix-lifecycle")
	})

	runner := &mockRunner{}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), "test-nix-lifecycle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Installed {
		t.Error("nix package should not be marked as installed")
	}
	if result.Error == "" {
		t.Error("expected advisory message for nix package")
	}
	if len(runner.commands) != 0 {
		t.Errorf("expected no commands for nix package, got %d", len(runner.commands))
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serverName string
		wantCmd    []string
		wantUpdate bool
	}{
		{
			name:       "UvTool",
			serverName: "man-pages",
			wantCmd:    []string{"uv", "tool", "upgrade", "man-mcp-server"},
			wantUpdate: true,
		},
		{
			name:       "NpmGlobal",
			serverName: "context7",
			wantCmd:    []string{"npm", "update", "-g", "@upstash/context7-mcp"},
			wantUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := &mockRunner{}
			state := &types.GeneratedState{
				McpServers: make(map[string]types.McpServerState),
			}
			lc := newTestLifecycle(runner, state)

			result, err := lc.Update(context.Background(), tt.serverName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Updated != tt.wantUpdate {
				t.Errorf("Updated = %v, want %v", result.Updated, tt.wantUpdate)
			}

			// A successful update runs the upgrade command first, then a
			// version-resolution probe, so assert on the first command.
			if len(runner.commands) == 0 {
				t.Fatal("expected at least 1 command, got 0")
			}
			got := runner.commands[0]
			if len(got) != len(tt.wantCmd) {
				t.Fatalf("command = %v, want %v", got, tt.wantCmd)
			}
			for i := range got {
				if got[i] != tt.wantCmd[i] {
					t.Errorf("command[%d] = %q, want %q", i, got[i], tt.wantCmd[i])
				}
			}
		})
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()

	t.Run("UvTool", func(t *testing.T) {
		t.Parallel()

		runner := &mockRunner{}
		state := &types.GeneratedState{
			McpServers: make(map[string]types.McpServerState),
		}
		lc := newTestLifecycle(runner, state)

		result, err := lc.Remove(context.Background(), "man-pages")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Removed {
			t.Error("expected Removed = true")
		}

		wantCmd := []string{"uv", "tool", "uninstall", "man-mcp-server"}
		if len(runner.commands) != 1 {
			t.Fatalf("expected 1 command, got %d", len(runner.commands))
		}
		got := runner.commands[0]
		for i := range got {
			if got[i] != wantCmd[i] {
				t.Errorf("command[%d] = %q, want %q", i, got[i], wantCmd[i])
			}
		}
	})

	t.Run("ClearsState", func(t *testing.T) {
		t.Parallel()

		runner := &mockRunner{}
		state := &types.GeneratedState{
			McpServers: make(map[string]types.McpServerState),
		}
		lc := newTestLifecycle(runner, state)

		// Install first.
		_, err := lc.Install(context.Background(), "man-pages")
		if err != nil {
			t.Fatalf("install failed: %v", err)
		}
		if _, ok := state.McpServers["man-pages"]; !ok {
			t.Fatal("expected man-pages in state after install")
		}

		// Remove.
		_, err = lc.Remove(context.Background(), "man-pages")
		if err != nil {
			t.Fatalf("remove failed: %v", err)
		}
		if _, ok := state.McpServers["man-pages"]; ok {
			t.Error("expected man-pages removed from state after removal")
		}
	})
}

// TestInstall_FailedCommandDoesNotWriteState guards the fail-closed contract:
// when the package-manager install command fails, the server must NOT be
// recorded as installed in state. (Before the fix this wrote a bogus
// "installed"/"latest" entry.)
func TestInstall_FailedCommandDoesNotWriteState(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{err: errors.New("boom: exit status 1")}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), "man-pages")
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Installed {
		t.Error("failed install must not report Installed = true")
	}
	if result.Error == "" {
		t.Error("expected result.Error to describe the failure")
	}
	if _, ok := state.McpServers["man-pages"]; ok {
		t.Error("failed install must NOT write success state")
	}
	// Only the install command should have run; the version probe must be
	// skipped once the install fails.
	if len(runner.commands) != 1 {
		t.Errorf("expected exactly 1 command (install only), got %d: %v", len(runner.commands), runner.commands)
	}
}

// TestInstall_RecordsResolvedVersionUvTool asserts a successful uv install
// records the real version parsed from `uv tool list`, not the hardcoded
// "latest", and marks health as verified-installed.
func TestInstall_RecordsResolvedVersionUvTool(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "uv" && len(args) >= 2 && args[0] == "tool" && args[1] == "list" {
				return []byte("man-mcp-server v1.4.2\n- man-mcp-server\nother-tool v9.9.9\n- other-tool\n"), nil
			}
			return nil, nil // install succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), "man-pages")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Installed {
		t.Fatal("expected Installed = true")
	}
	if result.Version != "1.4.2" {
		t.Errorf("result.Version = %q, want %q", result.Version, "1.4.2")
	}

	st, ok := state.McpServers["man-pages"]
	if !ok {
		t.Fatal("expected man-pages recorded in state")
	}
	if st.InstalledVersion != "1.4.2" {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, "1.4.2")
	}
	if st.InstalledVersion == "latest" {
		t.Error("version must not be the hardcoded 'latest'")
	}
	if st.LastHealthStatus != "installed" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "installed")
	}
}

// TestInstall_RecordsResolvedVersionNpm asserts a successful npm install records
// the real version parsed from `npm ls -g --json`.
func TestInstall_RecordsResolvedVersionNpm(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "npm" && len(args) >= 1 && args[0] == "ls" {
				return []byte(`{"dependencies":{"@madhan-g-p/devdocs-mcp-server":{"version":"2.0.1"}}}`), nil
			}
			return nil, nil // install succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), "local-docs-devdocs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != "2.0.1" {
		t.Errorf("result.Version = %q, want %q", result.Version, "2.0.1")
	}
	st := state.McpServers["local-docs-devdocs"]
	if st.InstalledVersion != "2.0.1" {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, "2.0.1")
	}
	if st.LastHealthStatus != "installed" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "installed")
	}
}

// TestInstall_UnverifiedWhenPackageNotFound asserts that when the install
// command succeeds but the package cannot be found in the manager inventory,
// the version is recorded as unknown and health as unverified (an honest
// unknown rather than a fake "latest"/"installed").
func TestInstall_UnverifiedWhenPackageNotFound(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "uv" && len(args) >= 2 && args[1] == "list" {
				return []byte("some-other-tool v1.0.0\n- some-other-tool\n"), nil
			}
			return nil, nil // install succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), "man-pages")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != "unknown" {
		t.Errorf("result.Version = %q, want %q", result.Version, "unknown")
	}
	st := state.McpServers["man-pages"]
	if st.InstalledVersion != "unknown" {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, "unknown")
	}
	if st.LastHealthStatus != "unverified" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "unverified")
	}
}

// TestUpdate_FailedCommandDoesNotWriteState guards the fail-closed contract for
// Update: a failed upgrade must not overwrite existing state with a bogus
// success entry.
func TestUpdate_FailedCommandDoesNotWriteState(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{err: errors.New("boom: exit status 1")}
	existing := types.McpServerState{InstalledVersion: "1.0.0", LastHealthStatus: "installed"}
	state := &types.GeneratedState{
		McpServers: map[string]types.McpServerState{"man-pages": existing},
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Update(context.Background(), "man-pages")
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Updated {
		t.Error("failed update must not report Updated = true")
	}
	if result.Error == "" {
		t.Error("expected result.Error to describe the failure")
	}
	// Existing state must be untouched by a failed upgrade.
	got := state.McpServers["man-pages"]
	if got.InstalledVersion != "1.0.0" {
		t.Errorf("failed update mutated state: InstalledVersion = %q, want %q", got.InstalledVersion, "1.0.0")
	}
	// Only the upgrade command should have run; no version probe.
	if len(runner.commands) != 1 {
		t.Errorf("expected exactly 1 command (upgrade only), got %d: %v", len(runner.commands), runner.commands)
	}
}

// TestUpdate_RecordsResolvedVersion asserts a successful upgrade records the
// real resolved version rather than "latest".
func TestUpdate_RecordsResolvedVersion(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "uv" && len(args) >= 2 && args[1] == "list" {
				return []byte("man-mcp-server v2.5.0\n- man-mcp-server\n"), nil
			}
			return nil, nil // upgrade succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Update(context.Background(), "man-pages")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NewVersion != "2.5.0" {
		t.Errorf("result.NewVersion = %q, want %q", result.NewVersion, "2.5.0")
	}
	if result.NewVersion == "latest" {
		t.Error("NewVersion must not be the hardcoded 'latest'")
	}
	st := state.McpServers["man-pages"]
	if st.InstalledVersion != "2.5.0" {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, "2.5.0")
	}
	if st.LastHealthStatus != "installed" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "installed")
	}
}
