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
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestPlanProbes_GeneratedConfigIsTrusted guards F095 against false
// "not-probed" results: every server definition qsdev itself writes to
// .mcp.json, including derived variants such as semble with text-file
// indexing, matches a trusted definition. Such entries may still be skipped by
// the launcher rule, which no override lifts, but never as untrusted.
func TestPlanProbes_GeneratedConfigIsTrusted(t *testing.T) {
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
				override, err := sembleTextFilesServer(entry, def)
				if err != nil {
					t.Fatal(err)
				}
				cfg.MCPServers = append(append([]MCPServerConfig{}, cfg.MCPServers...), override)
			}
			f, err := GenerateMcpJson(types.WizardAnswers{MCPServers: cat.MCPServerNames(), ProjectRoot: root}, cfg)
			if err != nil {
				t.Fatal(err)
			}
			var generated McpJSON
			if err := json.Unmarshal(f.Content, &generated); err != nil {
				t.Fatal(err)
			}
			servers := make([]mcphealth.ServerConfig, 0, len(generated.MCPServers))
			for name, e := range generated.MCPServers {
				servers = append(servers, mcphealth.ServerConfig{Name: name, Command: e.Command, Args: e.Args, URL: e.URL, Env: e.Env, Headers: e.Headers})
			}
			_, skipped := mcpregistry.PlanProbes(servers, mcpregistry.TrustedDefinitions(configuredServerSpecs()), false)
			for _, s := range skipped {
				if s.Overridable {
					t.Errorf("generated server %s treated as untrusted: %s", s.Name, s.Reason)
				}
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
		if !strings.Contains(out, mcphealth.StatusNotProbed) || !strings.Contains(out, "--probe-untrusted") {
			t.Errorf("expected a not-probed entry naming --probe-untrusted, got:\n%s", out)
		}
	}

	_, _ = runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe-untrusted")
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("--probe-untrusted should run the command: %v", err)
	}
}

// requestRecorder is an HTTP endpoint that records the Authorization header
// and query of every request it receives and answers 404.
type requestRecorder struct {
	mu      sync.Mutex
	auth    []string
	queries []string
}

func newRequestRecorder(t *testing.T) (*requestRecorder, string) {
	t.Helper()
	rec := &requestRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.auth = append(rec.auth, r.Header.Get("Authorization"))
		rec.queries = append(rec.queries, r.URL.RawQuery)
		rec.mu.Unlock()
		http.Error(w, "no", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return rec, srv.URL
}

// snapshot returns copies of the recorded Authorization headers and queries.
func (r *requestRecorder) snapshot() (auth, queries []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.auth), slices.Clone(r.queries)
}

// writeMCPJSON writes content as the .mcp.json of a new project directory.
func writeMCPJSON(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// assertUntrustedEndpointLiteral runs status and health against an untrusted
// remote entry using secretVar and checks that the endpoint is not contacted
// by default, and that under --probe-untrusted it receives the reference
// unexpanded.
func assertUntrustedEndpointLiteral(t *testing.T, secretVar string) {
	t.Helper()
	rec, base := newRequestRecorder(t)
	t.Setenv(secretVar, "s3cr3t")
	dir := writeMCPJSON(t, `{"mcpServers":{"h":{"type":"http","url":"`+base+`/mcp?leak=${`+secretVar+`}","headers":{"Authorization":"Bearer ${`+secretVar+`}"}}}}`)

	for name, newCmd := range map[string]func() *cobra.Command{"status": mcpStatusCmd, "health": mcpHealthCmd} {
		out, _ := runMCPSubcommand(t, dir, newCmd())
		if auth, _ := rec.snapshot(); len(auth) != 0 {
			t.Fatalf("%s dialed an untrusted endpoint (%d requests):\n%s", name, len(auth), out)
		}
		for _, want := range []string{mcphealth.StatusNotProbed, "untrusted remote endpoint", "--probe-untrusted"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s output missing %q:\n%s", name, want, out)
			}
		}
	}

	_, _ = runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe-untrusted")
	auth, queries := rec.snapshot()
	if len(auth) == 0 {
		t.Fatal("--probe-untrusted should dial the endpoint")
	}
	for i := range auth {
		if want := "Bearer ${" + secretVar + "}"; auth[i] != want {
			t.Errorf("Authorization = %q, want the literal %q", auth[i], want)
		}
		if strings.Contains(queries[i], "s3cr3t") {
			t.Errorf("query %q leaked the secret", queries[i])
		}
	}
}

// TestMCPStatus_UntrustedHTTPNotDialed is the U21-V01 regression: a remote
// entry from .mcp.json matching no trusted definition is not dialed by default,
// and under --probe-untrusted it receives its ${VAR} references literally, so
// repository content cannot make the probe send host secrets to it.
func TestMCPStatus_UntrustedHTTPNotDialed(t *testing.T) {
	assertUntrustedEndpointLiteral(t, "U21_T")
}

// TestMCPStatus_UntrustedURLNoSecret is the U22-WS1 acceptance form of the
// same guard, with the secret variable the plan names.
func TestMCPStatus_UntrustedURLNoSecret(t *testing.T) {
	assertUntrustedEndpointLiteral(t, "FAKE_SECRET_TOKEN")
}

// TestMCPStatus_ProbeUntrustedPlainHTTPRefused guards the U22-02 residual: even
// --probe-untrusted never dials plain http to a non-local host, and the skip
// does not suggest an override that cannot help.
func TestMCPStatus_ProbeUntrustedPlainHTTPRefused(t *testing.T) {
	dir := writeMCPJSON(t, `{"mcpServers":{"h":{"type":"http","url":"http://example.invalid/mcp"}}}`)
	for _, args := range [][]string{nil, {"--probe-untrusted"}} {
		out, err := runMCPSubcommand(t, dir, mcpStatusCmd(), args...)
		if err != nil {
			t.Fatalf("status %v: %v", args, err)
		}
		for _, want := range []string{mcphealth.StatusNotProbed, "plain http to a non-local host"} {
			if !strings.Contains(out, want) {
				t.Errorf("status %v output missing %q:\n%s", args, want, out)
			}
		}
		for _, unwanted := range []string{"dial tcp", "lookup", "--probe-untrusted"} {
			if strings.Contains(out, unwanted) {
				t.Errorf("status %v output contains %q:\n%s", args, unwanted, out)
			}
		}
	}
}

// TestMCPStatus_TrustedLauncherNotStarted is the U22-08 regression: the
// catalog's own context7 entry runs through npx, which would download and run
// the package, so no probe starts it, with or without --probe-untrusted, and
// the reason points at `qsdev mcp install`.
func TestMCPStatus_TrustedLauncherNotStarted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub npx is a POSIX shell script")
	}
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := cat.MCPServer("context7")
	if !ok {
		t.Fatal("catalog has no context7 server")
	}
	entry := catalogServerEntry("context7", def, nil)
	if entry.Command != "npx" {
		t.Fatalf("catalog context7 command = %q, want npx", entry.Command)
	}

	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "npx-ran")
	stub := "#!/bin/sh\ntouch '" + marker + "'\n"
	if err := os.WriteFile(filepath.Join(binDir, "npx"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	data, err := json.Marshal(McpJSON{MCPServers: map[string]MCPServerEntry{"context7": entry}})
	if err != nil {
		t.Fatal(err)
	}
	dir := writeMCPJSON(t, string(data))

	runs := []struct {
		newCmd func() *cobra.Command
		args   []string
	}{
		{mcpStatusCmd, nil},
		{mcpHealthCmd, nil},
		{mcpStatusCmd, []string{"--probe-untrusted"}},
	}
	for _, run := range runs {
		out, _ := runMCPSubcommand(t, dir, run.newCmd(), run.args...)
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s %v started the package launcher:\n%s", run.newCmd().Use, run.args, out)
		}
		for _, want := range []string{mcphealth.StatusNotProbed, "package launcher npx", "qsdev mcp install context7"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s %v output missing %q:\n%s", run.newCmd().Use, run.args, want, out)
			}
		}
		if strings.Contains(out, "--probe-untrusted") {
			t.Errorf("%s %v suggests --probe-untrusted, which cannot lift the launcher rule:\n%s", run.newCmd().Use, run.args, out)
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
