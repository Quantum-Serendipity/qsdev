package claudecode

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

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
