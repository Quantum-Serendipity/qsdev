package mcpregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

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

// isInventoryQuery reports whether a command lists installed packages
// (`uv tool list` or `npm ls`) rather than changing them.
func isInventoryQuery(name string, args []string) bool {
	return (name == "uv" && len(args) >= 2 && args[1] == "list") ||
		(name == "npm" && len(args) >= 1 && args[0] == "ls")
}

// inventoryOutput renders pkgs (name -> version) as the matching package
// manager's inventory listing.
func inventoryOutput(name string, pkgs map[string]string) []byte {
	if name == "uv" {
		var b strings.Builder
		for p, v := range pkgs {
			fmt.Fprintf(&b, "%s v%s\n- %s\n", p, v, p)
		}
		return []byte(b.String())
	}
	deps := make(map[string]map[string]string, len(pkgs))
	for p, v := range pkgs {
		deps[p] = map[string]string{"version": v}
	}
	out, _ := json.Marshal(map[string]any{"dependencies": deps})
	return out
}

// inventory returns a responder under which every package-manager command
// succeeds and the inventory listings report pkgs as installed.
func inventory(pkgs map[string]string) func(string, []string) ([]byte, error) {
	return func(name string, args []string) ([]byte, error) {
		if isInventoryQuery(name, args) {
			return inventoryOutput(name, pkgs), nil
		}
		return nil, nil
	}
}

// catalogPackages reports every installable registry server's package as
// installed at its pinned version.
var catalogPackages = func() map[string]string {
	pkgs := make(map[string]string)
	for _, def := range DefaultRegistry().All() {
		if def.PackageName != "" && def.Version != "" {
			pkgs[def.PackageName] = def.Version
		}
	}
	return pkgs
}()

// pinOf returns the version the registry pins for server.
func pinOf(t *testing.T, server string) string {
	t.Helper()
	def, ok := DefaultRegistry().ByName(server)
	if !ok || def.Version == "" {
		t.Fatalf("registry server %q has no pinned version", server)
	}
	return def.Version
}

// uvTestServer and npmTestServer are catalog servers installed through uv
// tool and npm global, whose pinned packages the tests install.
const (
	uvTestServer  = "mcp-nixos"
	uvTestPackage = "mcp-nixos"
	npmTestServer = "local-docs-devdocs"
)

func testStateLoader(state *types.GeneratedState) func() (*types.GeneratedState, error) {
	return func() (*types.GeneratedState, error) { return state, nil }
}

func testStateSaver(state *types.GeneratedState) func(*types.GeneratedState) error {
	return func(s *types.GeneratedState) error { *state = *s; return nil }
}

// testNow is the fixed clock tests run at, so release-age cutoffs are stable.
var testNow = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

const (
	testNpmCutoff = "2026-09-20T12:00:00Z" // testNow - npmMinReleaseAge
	testUvCutoff  = "2026-09-16T12:00:00Z" // testNow - uvMinReleaseAge
)

func newTestLifecycle(runner *mockRunner, state *types.GeneratedState) *McpLifecycle {
	return &McpLifecycle{
		CmdRunner:   runner,
		StateLoader: testStateLoader(state),
		StateSaver:  testStateSaver(state),
		Now:         func() time.Time { return testNow },
	}
}

func assertFirstCommand(t *testing.T, runner *mockRunner, want []string) {
	t.Helper()
	if len(runner.commands) == 0 {
		t.Fatal("expected at least 1 command, got 0")
	}
	if got := runner.commands[0]; !slices.Equal(got, want) {
		t.Errorf("command = %q, want %q", got, want)
	}
}

// TestInstall also covers F260: installs are age-gated and npm lifecycle
// scripts are disabled, since a global install bypasses the project .npmrc and
// the package-guard hook.
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
			serverName:  uvTestServer,
			wantCmd:     []string{"uv", "tool", "install", "--exclude-newer", testUvCutoff, uvTestPackage + "==" + pinOf(t, uvTestServer)},
			wantInstall: true,
		},
		{
			name:        "NpmGlobal",
			serverName:  npmTestServer,
			wantCmd:     []string{"npm", "install", "-g", "--ignore-scripts", "--before=" + testNpmCutoff, "@madhan-g-p/devdocs-mcp-server@" + pinOf(t, npmTestServer)},
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

			runner := &mockRunner{responder: inventory(catalogPackages)}
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
				t.Errorf("Installed = %v, want %v (error=%q)", result.Installed, tt.wantInstall, result.Error)
			}

			// A successful install runs the install command first, then a
			// version-resolution probe, so assert on the first command.
			assertFirstCommand(t, runner, tt.wantCmd)
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
			serverName: uvTestServer,
			wantCmd:    []string{"uv", "tool", "install", "--exclude-newer", testUvCutoff, uvTestPackage + "==" + pinOf(t, uvTestServer)},
			wantUpdate: true,
		},
		{
			name:       "NpmGlobal",
			serverName: "context7",
			wantCmd:    []string{"npm", "install", "-g", "--ignore-scripts", "--before=" + testNpmCutoff, "@upstash/context7-mcp@" + pinOf(t, "context7")},
			wantUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := &mockRunner{responder: inventory(catalogPackages)}
			state := &types.GeneratedState{
				McpServers: map[string]types.McpServerState{tt.serverName: {InstalledVersion: "0.9.0"}},
			}
			lc := newTestLifecycle(runner, state)

			result, err := lc.Update(context.Background(), tt.serverName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Updated != tt.wantUpdate {
				t.Errorf("Updated = %v, want %v (error=%q)", result.Updated, tt.wantUpdate, result.Error)
			}

			// A successful update runs the upgrade command first, then a
			// version-resolution probe, so assert on the first command.
			assertFirstCommand(t, runner, tt.wantCmd)
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

		result, err := lc.Remove(context.Background(), uvTestServer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Removed {
			t.Error("expected Removed = true")
		}

		wantCmd := []string{"uv", "tool", "uninstall", uvTestPackage}
		if len(runner.commands) != 1 {
			t.Fatalf("expected 1 command, got %d", len(runner.commands))
		}
		assertFirstCommand(t, runner, wantCmd)
	})

	t.Run("ClearsState", func(t *testing.T) {
		t.Parallel()

		runner := &mockRunner{responder: inventory(catalogPackages)}
		state := &types.GeneratedState{
			McpServers: make(map[string]types.McpServerState),
		}
		lc := newTestLifecycle(runner, state)

		// Install first.
		_, err := lc.Install(context.Background(), uvTestServer)
		if err != nil {
			t.Fatalf("install failed: %v", err)
		}
		if _, ok := state.McpServers[uvTestServer]; !ok {
			t.Fatal("expected server in state after install")
		}

		// Remove.
		_, err = lc.Remove(context.Background(), uvTestServer)
		if err != nil {
			t.Fatalf("remove failed: %v", err)
		}
		if _, ok := state.McpServers[uvTestServer]; ok {
			t.Error("expected server removed from state after removal")
		}
	})
}

// TestRemove_FailedUninstall is the F273 regression: a failed uninstall used
// to drop the state entry anyway, orphaning a still-installed package from
// every future `mcp update --all`. The entry must survive while the package is
// still installed, and go only once the package is confirmed absent.
func TestRemove_FailedUninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		stillPresent    bool
		inventoryBroken bool
		wantStateKept   bool
	}{
		{name: "package still installed keeps state", stillPresent: true, wantStateKept: true},
		{name: "package already gone drops stale state", stillPresent: false, wantStateKept: false},
		{name: "unreadable inventory keeps state", inventoryBroken: true, wantStateKept: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			present := map[string]string{}
			if tt.stillPresent {
				present[uvTestPackage] = "1.0.0"
			}
			runner := &mockRunner{responder: func(name string, args []string) ([]byte, error) {
				if isInventoryQuery(name, args) && !tt.inventoryBroken {
					return inventoryOutput(name, present), nil
				}
				return []byte("EACCES"), errors.New("exit status 1")
			}}
			state := &types.GeneratedState{
				McpServers: map[string]types.McpServerState{uvTestServer: {InstalledVersion: "1.0.0"}},
			}
			lc := newTestLifecycle(runner, state)

			result, err := lc.Remove(context.Background(), uvTestServer)
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if result.Removed || result.Error == "" {
				t.Errorf("Removed = %v, Error = %q; want a reported failure", result.Removed, result.Error)
			}
			if _, kept := state.McpServers[uvTestServer]; kept != tt.wantStateKept {
				t.Errorf("state entry kept = %v, want %v", kept, tt.wantStateKept)
			}
		})
	}
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

	result, err := lc.Install(context.Background(), uvTestServer)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Installed {
		t.Error("failed install must not report Installed = true")
	}
	if result.Error == "" {
		t.Error("expected result.Error to describe the failure")
	}
	if _, ok := state.McpServers[uvTestServer]; ok {
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

	pin := pinOf(t, uvTestServer)
	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "uv" && len(args) >= 2 && args[0] == "tool" && args[1] == "list" {
				return []byte(uvTestPackage + " v" + pin + "\n- " + uvTestPackage + "\nother-tool v9.9.9\n- other-tool\n"), nil
			}
			return nil, nil // install succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), uvTestServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Installed {
		t.Fatal("expected Installed = true")
	}
	if result.Version != pin {
		t.Errorf("result.Version = %q, want %q", result.Version, pin)
	}

	st, ok := state.McpServers[uvTestServer]
	if !ok {
		t.Fatal("expected server recorded in state")
	}
	if st.InstalledVersion != pin {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, pin)
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

	pin := pinOf(t, npmTestServer)
	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "npm" && len(args) >= 1 && args[0] == "ls" {
				return []byte(`{"dependencies":{"@madhan-g-p/devdocs-mcp-server":{"version":"` + pin + `"}}}`), nil
			}
			return nil, nil // install succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), npmTestServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != pin {
		t.Errorf("result.Version = %q, want %q", result.Version, pin)
	}
	st := state.McpServers[npmTestServer]
	if st.InstalledVersion != pin {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, pin)
	}
	if st.LastHealthStatus != "installed" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "installed")
	}
}

// TestInstall_NotInstalledWhenPackageNotFound is the F272 regression: when the
// install command exits 0 but the package is absent from the manager's
// inventory, the server is not installed. It must not be reported as Installed
// nor recorded in state.
func TestInstall_NotInstalledWhenPackageNotFound(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{responder: inventory(map[string]string{"some-other-tool": "1.0.0"})}
	state := &types.GeneratedState{
		McpServers: make(map[string]types.McpServerState),
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), uvTestServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Installed {
		t.Error("Installed = true for a package missing from the inventory")
	}
	if !strings.Contains(result.Error, "not in its inventory") {
		t.Errorf("Error = %q, want it to explain the package is missing", result.Error)
	}
	if _, ok := state.McpServers[uvTestServer]; ok {
		t.Error("an unverified install must not be recorded in state")
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
		McpServers: map[string]types.McpServerState{uvTestServer: existing},
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Update(context.Background(), uvTestServer)
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
	got := state.McpServers[uvTestServer]
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

	pin := pinOf(t, uvTestServer)
	runner := &mockRunner{
		responder: func(name string, args []string) ([]byte, error) {
			if name == "uv" && len(args) >= 2 && args[1] == "list" {
				return []byte(uvTestPackage + " v" + pin + "\n- " + uvTestPackage + "\n"), nil
			}
			return nil, nil // upgrade succeeds
		},
	}
	state := &types.GeneratedState{
		McpServers: map[string]types.McpServerState{uvTestServer: {InstalledVersion: "2.4.0"}},
	}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Update(context.Background(), uvTestServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PreviousVer != "2.4.0" {
		t.Errorf("result.PreviousVer = %q, want %q", result.PreviousVer, "2.4.0")
	}
	if result.NewVersion != pin {
		t.Errorf("result.NewVersion = %q, want %q", result.NewVersion, pin)
	}
	if result.NewVersion == "latest" {
		t.Error("NewVersion must not be the hardcoded 'latest'")
	}
	st := state.McpServers[uvTestServer]
	if st.InstalledVersion != pin {
		t.Errorf("state InstalledVersion = %q, want %q", st.InstalledVersion, pin)
	}
	if st.LastHealthStatus != "installed" {
		t.Errorf("state LastHealthStatus = %q, want %q", st.LastHealthStatus, "installed")
	}
}

// TestUpdate_UnverifiedOutcomes is the F272 regression for Update: updating a
// server qsdev never installed (`npm update -g` exits 0 for it) or whose
// package vanished must not report Updated nor create a state entry.
func TestUpdate_UnverifiedOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tracked   bool
		wantCmds  int
		wantError string
	}{
		{name: "never installed", tracked: false, wantCmds: 0, wantError: "not installed"},
		{name: "missing after upgrade", tracked: true, wantCmds: 2, wantError: "not in its inventory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runner := &mockRunner{responder: inventory(nil)}
			state := &types.GeneratedState{McpServers: map[string]types.McpServerState{}}
			if tt.tracked {
				state.McpServers["context7"] = types.McpServerState{InstalledVersion: "1.0.0"}
			}
			lc := newTestLifecycle(runner, state)

			result, err := lc.Update(context.Background(), "context7")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Updated {
				t.Error("Updated = true for an unverified update")
			}
			if !strings.Contains(result.Error, tt.wantError) {
				t.Errorf("Error = %q, want it to contain %q", result.Error, tt.wantError)
			}
			if len(runner.commands) != tt.wantCmds {
				t.Errorf("ran %d commands, want %d: %v", len(runner.commands), tt.wantCmds, runner.commands)
			}
			if st, ok := state.McpServers["context7"]; ok != tt.tracked || (ok && st.InstalledVersion != "1.0.0") {
				t.Errorf("state entry = %+v (present %v), want it unchanged", st, ok)
			}
		})
	}
}

// TestInstall_RefusesUnpinnedPackage covers F082: a registry server without
// an exact pinned version is never installed, since the package manager would
// take whatever release the registry serves.
func TestInstall_RefusesUnpinnedPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  McpInstallMethod
		version string
	}{
		{name: "uv no version", method: InstallUvTool},
		{name: "uv wildcard", method: InstallUvTool, version: "1.*"},
		{name: "npm no version", method: InstallNpmGlobal},
		{name: "npm range", method: InstallNpmGlobal, version: "^1.2.0"},
		{name: "npm dist-tag", method: InstallNpmGlobal, version: "latest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := "test-unpinned-" + strings.ReplaceAll(tt.name, " ", "-")
			reg := DefaultRegistry()
			if err := reg.Register(McpServerDefinition{
				Name:          server,
				Command:       "test",
				Transport:     TransportStdio,
				Source:        SourceBuiltin,
				InstallMethod: tt.method,
				PackageName:   "test-pkg",
				Version:       tt.version,
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { reg.Delete(server) })

			runner := &mockRunner{responder: inventory(map[string]string{"test-pkg": "1.0.0"})}
			state := &types.GeneratedState{McpServers: map[string]types.McpServerState{server: {InstalledVersion: "1.0.0"}}}
			lc := newTestLifecycle(runner, state)

			install, err := lc.Install(context.Background(), server)
			if err != nil {
				t.Fatal(err)
			}
			update, err := lc.Update(context.Background(), server)
			if err != nil {
				t.Fatal(err)
			}
			if install.Installed || update.Updated {
				t.Errorf("Installed = %v, Updated = %v for an unpinned package", install.Installed, update.Updated)
			}
			for _, msg := range []string{install.Error, update.Error} {
				if !strings.Contains(msg, "no exact pinned version") {
					t.Errorf("Error = %q, want it to explain the missing pin", msg)
				}
			}
			if len(runner.commands) != 0 {
				t.Errorf("ran %v for an unpinned package", runner.commands)
			}
		})
	}
}

// TestInstall_RefusesOtherRelease covers F082: an install that leaves a
// release other than the pinned one is not recorded, so .mcp.json never
// prefers a binary qsdev did not vouch for.
func TestInstall_RefusesOtherRelease(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{responder: inventory(map[string]string{uvTestPackage: "0.0.1"})}
	state := &types.GeneratedState{McpServers: map[string]types.McpServerState{}}
	lc := newTestLifecycle(runner, state)

	result, err := lc.Install(context.Background(), uvTestServer)
	if err != nil {
		t.Fatal(err)
	}
	if result.Installed {
		t.Error("Installed = true for a release other than the pin")
	}
	if !strings.Contains(result.Error, "not the pinned "+pinOf(t, uvTestServer)) {
		t.Errorf("Error = %q, want it to name the pinned release", result.Error)
	}
	if _, ok := state.McpServers[uvTestServer]; ok {
		t.Error("a release other than the pin must not be recorded in state")
	}
}
