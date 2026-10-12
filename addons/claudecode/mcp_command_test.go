package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/agentpostmortem"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/versionsentinel"
)

func TestMcpCmd_Structure(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()
	if cmd.Use != "mcp" {
		t.Errorf("mcpCmd().Use = %q, want %q", cmd.Use, "mcp")
	}
	if cmd.Short == "" {
		t.Error("mcpCmd().Short is empty")
	}
}

// TestMcpCmd_ServesModulesOnlyThroughServe is the F525 regression: the tool
// modules (agent-postmortem, version-sentinel) are served by `mcp serve
// --module`, behind the universal server's middleware. The `mcp <module>`
// subcommands an older .mcp.json launches survive only as hidden aliases of it
// (see mcpserve.LegacyModuleCommands), never as bespoke servers.
func TestMcpCmd_ServesModulesOnlyThroughServe(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()
	serve, _, err := cmd.Find([]string{"serve"})
	if err != nil || serve.Use != "serve" {
		t.Fatalf("finding serve subcommand: %v", err)
	}
	if serve.Flags().Lookup("module") == nil {
		t.Error("serve has no --module flag")
	}
	for _, module := range []string{agentpostmortem.ModuleName, versionsentinel.ModuleName} {
		sub, _, err := cmd.Find([]string{module})
		if err != nil || sub.Name() != module {
			t.Errorf("mcp %s: not found (%v); an older .mcp.json launches it", module, err)
			continue
		}
		if !sub.Hidden {
			t.Errorf("mcp %s is a visible subcommand; it must be a hidden alias of mcp serve --module", module)
		}
	}
}

func TestMcpCmd_HasDiagnosticSubcommands(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()

	for _, name := range []string{"status", "list"} {
		sub, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %s subcommand: %v", name, err)
		}
		if sub.Use != name {
			t.Errorf("subcommand Use = %q, want %q", sub.Use, name)
		}
	}
}

// localFileArgFixture creates <root>/server.js and <root>/sub, changes into
// sub and returns root and sub.
func localFileArgFixture(t *testing.T) (root, sub string) {
	t.Helper()
	root = t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "server.js"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub = filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	return root, sub
}

type localFileArgCase struct {
	arg, dir string
	want     string
	wantOK   bool
}

func checkLocalFileArg(t *testing.T, tests []localFileArgCase) {
	t.Helper()
	for _, tt := range tests {
		got, ok := localFileArg(tt.arg, tt.dir)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("localFileArg(%q, %q) = %q, %v; want %q, %v", tt.arg, tt.dir, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestLocalFileArg_RelativeToProjectRoot is the U16-10 regression: a relative
// script argument is the file the server loads from the project root it is
// launched in, whatever directory qsdev runs from.
func TestLocalFileArg_RelativeToProjectRoot(t *testing.T) {
	root, sub := localFileArgFixture(t)
	want := filepath.Join(root, "server.js")
	checkLocalFileArg(t, []localFileArgCase{
		{arg: "server.js", dir: root, want: want, wantOK: true},
		{arg: "./server.js", dir: root, want: want, wantOK: true},
		{arg: "--require=./server.js", dir: root, want: want, wantOK: true},
		{arg: want, dir: sub, want: want, wantOK: true},
		{arg: "server.js", dir: sub, wantOK: false},
		{arg: "--flag", dir: root, wantOK: false},
	})
}

// TestLocalFileArg_EmptyDirUsesCwd pins the base behaviour: with no dir, a
// relative argument resolves against the current directory.
func TestLocalFileArg_EmptyDirUsesCwd(t *testing.T) {
	root, sub := localFileArgFixture(t)
	if err := os.WriteFile(filepath.Join(sub, "local.js"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	checkLocalFileArg(t, []localFileArgCase{
		{arg: "local.js", dir: "", want: filepath.Join(sub, "local.js"), wantOK: true},
		{arg: "../server.js", dir: "", want: filepath.Join(root, "server.js"), wantOK: true},
		{arg: "server.js", dir: "", wantOK: false},
	})
}
