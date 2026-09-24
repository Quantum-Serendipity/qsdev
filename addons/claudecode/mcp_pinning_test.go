package claudecode

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeInstalledMCPState records servers as installed in the claude state
// file under root, as `qsdev mcp install` does.
func writeInstalledMCPState(t *testing.T, root string, servers map[string]types.McpServerState) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(statePath()))
	s := types.GeneratedState{Files: map[string]types.FileState{}, McpServers: servers}
	if err := state.SaveStateToFile(path, s); err != nil {
		t.Fatalf("writing state: %v", err)
	}
}

// installedAtPin returns an install record at the pinned release for every
// catalog server whose install method provides a binary.
func installedAtPin(cat *catalog.Catalog) map[string]types.McpServerState {
	out := make(map[string]types.McpServerState)
	for name, def := range cat.MCPServers() {
		if def.Bin != "" {
			out[name] = types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}
		}
	}
	return out
}

func generatedEntries(t *testing.T, answers types.WizardAnswers, cfg Config) map[string]MCPServerEntry {
	t.Helper()
	f, err := GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("GenerateMcpJson: %v", err)
	}
	var mcp McpJSON
	if err := json.Unmarshal(f.Content, &mcp); err != nil {
		t.Fatalf("parsing .mcp.json: %v", err)
	}
	return mcp.MCPServers
}

// TestGenerateMcpJson_RefusesUnpinnedLaunchers covers F082: a server that
// would fetch a package at launch without an exact version is refused,
// whichever source configured it.
func TestGenerateMcpJson_RefusesUnpinnedLaunchers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{name: "npx auto-confirm latest", command: "npx", args: []string{"-y", "@upstash/context7-mcp"}, wantErr: true},
		{name: "npx range", command: "npx", args: []string{"some-mcp@^1.2.0"}, wantErr: true},
		{name: "npx dist-tag", command: "npx", args: []string{"some-mcp@latest"}, wantErr: true},
		{name: "uvx unpinned from", command: "uvx", args: []string{"--from", "semble[mcp]", "semble"}, wantErr: true},
		{name: "uvx bare package", command: "uvx", args: []string{"mcp-nixos"}, wantErr: true},
		{name: "uvx wildcard", command: "uvx", args: []string{"--from", "pkg==1.*", "pkg"}, wantErr: true},
		{name: "absolute npx path", command: "/usr/bin/npx", args: []string{"some-mcp"}, wantErr: true},
		{name: "env wrapped uvx", command: "env", args: []string{"FOO=1", "uvx", "pkg"}, wantErr: true},
		{name: "pnpm dlx", command: "pnpm", args: []string{"dlx", "some-mcp"}, wantErr: true},
		{name: "docker without digest", command: "docker", args: []string{"run", "-i", "mcp/server:latest"}, wantErr: true},
		{name: "npx exact", command: "npx", args: []string{"some-mcp@1.2.3"}},
		{name: "uvx exact", command: "uvx", args: []string{"--from", "semble[mcp]==0.6.0", "semble"}},
		{name: "local binary", command: "my-mcp-server", args: []string{"--stdio"}},
		{name: "pnpm local script", command: "pnpm", args: []string{"exec", "tsx", "server.ts"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := NewConfig(WithMCPServer(MCPServerConfig{Name: "custom", Command: tt.command, Args: tt.args}))
			_, err := GenerateMcpJson(types.WizardAnswers{}, cfg)
			if got := errors.Is(err, ErrUnpinnedMCPServer); got != tt.wantErr {
				t.Fatalf("err = %v, want ErrUnpinnedMCPServer: %v", err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestCatalogMCPServers_Pinned covers F082 for the catalog: every server
// generates (so no launcher is unpinned), and a server with an install
// method pins the same exact release in its launcher and for `qsdev mcp
// install`.
func TestCatalogMCPServers_Pinned(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	entries := generatedEntries(t, types.WizardAnswers{MCPServers: cat.MCPServerNames()}, Config{})
	for name, def := range cat.MCPServers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, ok := entries[name]; !ok {
				t.Fatalf("%s missing from generated .mcp.json", name)
			}
			var spec string
			switch def.InstallMethod {
			case "npm-global":
				spec = def.PackageName + "@" + def.Version
			case "uv-tool":
				spec = def.PackageName + "==" + def.Version
			default:
				return
			}
			if def.Version == "" || def.Bin == "" {
				t.Fatalf("install method %s without version (%q) or bin (%q)", def.InstallMethod, def.Version, def.Bin)
			}
			if def.Command != def.Bin && !slices.Contains(def.Args, spec) {
				t.Errorf("launcher %s %v does not run the pinned %s", def.Command, def.Args, spec)
			}
			if def.Command != def.Bin && !slices.Equal(def.Args[len(def.Args)-len(def.BinArgs):], def.BinArgs) {
				t.Errorf("launcher args %v do not end with bin_args %v", def.Args, def.BinArgs)
			}
		})
	}
}

// TestGenerateMcpJson_PrefersInstalledBinary covers F082: a server `qsdev mcp
// install` installed at the pinned release runs the installed binary; any
// other record keeps the pinned launcher.
func TestGenerateMcpJson_PrefersInstalledBinary(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	const server = "postgres" // its binary takes arguments (bin_args)
	def, ok := cat.MCPServer(server)
	if !ok || def.Bin == "" {
		t.Fatalf("catalog server %s must provide a binary", server)
	}
	launcher := catalogDefToEntry(def)
	binary := MCPServerEntry{Command: def.Bin, Args: def.BinArgs, Env: def.Env}

	tests := []struct {
		name   string
		record *types.McpServerState
		noRoot bool
		want   MCPServerEntry
	}{
		{name: "installed at pin", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}, want: binary},
		{name: "installed at another version", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: "0.0.1"}, want: launcher},
		{name: "installed by another method", record: &types.McpServerState{InstallMethod: "npm-global", InstalledVersion: def.Version}, want: launcher},
		{name: "not installed", want: launcher},
		{name: "no project root", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}, noRoot: true, want: launcher},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.record != nil {
				writeInstalledMCPState(t, root, map[string]types.McpServerState{server: *tt.record})
			}
			answers := types.WizardAnswers{MCPServers: []string{server}, ProjectRoot: root}
			if tt.noRoot {
				answers.ProjectRoot = ""
			}
			got := generatedEntries(t, answers, Config{})[server]
			if got.Command != tt.want.Command || !slices.Equal(got.Args, tt.want.Args) {
				t.Errorf("entry = %s %v, want %s %v", got.Command, got.Args, tt.want.Command, tt.want.Args)
			}

			// `qsdev enable` writes the same entry into .mcp.json.
			raw, err := mcpServerContentFunc(server)(answers)
			if err != nil {
				t.Fatalf("enable content: %v", err)
			}
			var enabled MCPServerEntry
			if err := json.Unmarshal(raw, &enabled); err != nil {
				t.Fatal(err)
			}
			if enabled.Command != tt.want.Command || !slices.Equal(enabled.Args, tt.want.Args) {
				t.Errorf("enable entry = %s %v, want %s %v", enabled.Command, enabled.Args, tt.want.Command, tt.want.Args)
			}
		})
	}
}

// TestSembleTextFiles_InstalledBinary covers F082: text-file indexing appends
// its flag to whichever entry the server runs, so installing semble does not
// fall back to the fetch-on-run launcher.
func TestSembleTextFiles_InstalledBinary(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := cat.MCPServer(sembleServerName)
	if !ok {
		t.Fatal("semble missing from catalog")
	}
	root := t.TempDir()
	writeInstalledMCPState(t, root, installedAtPin(cat))
	answers := types.WizardAnswers{ProjectRoot: root}
	answers.AgentTools.SembleEnabled = true
	answers.AgentTools.SembleMode = "mcp"
	answers.AgentTools.SembleTextFiles = true

	res, err := generateSembleConfig(answers)
	if err != nil {
		t.Fatal(err)
	}
	if res.Override == nil {
		t.Fatal("no semble override for text-file indexing")
	}
	want := append(append([]string{}, def.BinArgs...), "--include-text-files")
	if res.Override.Command != def.Bin || !slices.Equal(res.Override.Args, want) {
		t.Errorf("override = %s %v, want %s %v", res.Override.Command, res.Override.Args, def.Bin, want)
	}
}
