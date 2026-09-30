package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

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
	for _, tc := range []struct {
		name      string
		installed bool
	}{
		{name: "pinned launchers"},
		{name: "installed binaries", installed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var root string
			if tc.installed {
				root = t.TempDir()
				writeInstalledMCPState(t, root, installedAtPin(cat))
			}
			cfg := addon.Config
			if def, ok := cat.MCPServer(sembleServerName); ok {
				entry := catalogServerEntry(sembleServerName, def, installedMCPServers(root))
				cfg.MCPServers = append(append([]MCPServerConfig{}, cfg.MCPServers...), sembleTextFilesServer(entry))
			}
			f, err := GenerateMcpJson(types.WizardAnswers{MCPServers: cat.MCPServerNames(), ProjectRoot: root}, cfg)
			if err != nil {
				t.Fatal(err)
			}
			var generated McpJSON
			if err := json.Unmarshal(f.Content, &generated); err != nil {
				t.Fatal(err)
			}
			servers := make(map[string]mcphealth.ServerConfig, len(generated.MCPServers))
			for name, e := range generated.MCPServers {
				servers[name] = mcphealth.ServerConfig{Name: name, Command: e.Command, Args: e.Args, URL: e.URL, Env: e.Env, Headers: e.Headers}
			}
			if _, skipped := mcpregistry.PartitionTrusted(servers, trustedMCPDefinitions()); len(skipped) > 0 {
				t.Errorf("generated servers treated as untrusted: %v", skipped)
			}
		})
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

// TestMCPStatus_UntrustedHTTPNotDialed is the U21-V01 regression: a remote
// entry from .mcp.json matching no trusted definition is not dialed by default,
// and under --probe-untrusted it receives its ${VAR} references literally, so
// repository content cannot make the probe send host secrets to it.
func TestMCPStatus_UntrustedHTTPNotDialed(t *testing.T) {
	var (
		mu       sync.Mutex
		hits     int
		gotAuth  []string
		gotQuery []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		gotQuery = append(gotQuery, r.URL.RawQuery)
		mu.Unlock()
		http.Error(w, "no", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("U21_T", "s3cr3t")

	dir := t.TempDir()
	mcp := `{"mcpServers":{"h":{"type":"http","url":"` + srv.URL + `/mcp?leak=${U21_T}","headers":{"Authorization":"Bearer ${U21_T}"}}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcp), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, newCmd := range map[string]func() *cobra.Command{"status": mcpStatusCmd, "health": mcpHealthCmd} {
		out, _ := runMCPSubcommand(t, dir, newCmd())
		mu.Lock()
		n := hits
		mu.Unlock()
		if n != 0 {
			t.Fatalf("%s dialed an untrusted endpoint (%d requests):\n%s", name, n, out)
		}
		for _, want := range []string{statusNotProbed, "untrusted remote endpoint", "--probe-untrusted"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s output missing %q:\n%s", name, want, out)
			}
		}
	}

	_, _ = runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe-untrusted")
	mu.Lock()
	defer mu.Unlock()
	if hits == 0 {
		t.Fatal("--probe-untrusted should dial the endpoint")
	}
	for i := range gotAuth {
		if gotAuth[i] != "Bearer ${U21_T}" {
			t.Errorf("Authorization = %q, want the literal template", gotAuth[i])
		}
		if strings.Contains(gotQuery[i], "s3cr3t") {
			t.Errorf("query %q leaked the secret", gotQuery[i])
		}
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
