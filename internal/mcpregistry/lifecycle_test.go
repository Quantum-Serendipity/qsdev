package mcpregistry

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type mockRunner struct {
	commands [][]string
	err      error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.commands = append(m.commands, append([]string{name}, args...))
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
			wantCmd:     []string{"npm", "install", "-g", "devdocs-mcp-server"},
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
				if len(runner.commands) != 1 {
					t.Fatalf("expected 1 command, got %d", len(runner.commands))
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

			if len(runner.commands) != 1 {
				t.Fatalf("expected 1 command, got %d", len(runner.commands))
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
