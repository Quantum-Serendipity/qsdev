package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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

// untrustedCommandFixture writes a project whose .mcp.json runs a
// repository-supplied shell command that creates the returned marker file.
func untrustedCommandFixture(t *testing.T) (dir, marker string) {
	t.Helper()
	marker = filepath.Join(t.TempDir(), "pwned")
	return writeMCPJSON(t, `{"mcpServers":{"x":{"command":"sh","args":["-c","touch `+marker+`"]}}}`), marker
}

// TestMCPProbe_UntrustedCommandNotRun guards F095: status and health must not
// execute a repository-supplied command, by default or under --probe, unless
// --probe-untrusted is given.
func TestMCPProbe_UntrustedCommandNotRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell as the untrusted command")
	}
	dir, marker := untrustedCommandFixture(t)

	for _, newCmd := range []func() *cobra.Command{mcpStatusCmd, mcpHealthCmd} {
		for _, args := range [][]string{nil, {"--probe"}} {
			out, _ := runMCPSubcommand(t, dir, newCmd(), args...)
			if fileExists(marker) {
				t.Fatalf("%s %v executed the untrusted command:\n%s", newCmd().Use, args, out)
			}
			if !strings.Contains(out, mcphealth.StatusNotProbed) || !strings.Contains(out, "--probe-untrusted") {
				t.Errorf("%s %v: expected a not-probed entry naming --probe-untrusted, got:\n%s", newCmd().Use, args, out)
			}
		}
	}

	_, _ = runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe-untrusted")
	if !fileExists(marker) {
		t.Error("--probe-untrusted should run the command")
	}
}

// fileExists reports whether path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
// by default or under --probe, and that under --probe-untrusted it receives
// the reference unexpanded.
func assertUntrustedEndpointLiteral(t *testing.T, secretVar string) {
	t.Helper()
	rec, base := newRequestRecorder(t)
	t.Setenv(secretVar, "s3cr3t")
	dir := writeMCPJSON(t, `{"mcpServers":{"h":{"type":"http","url":"`+base+`/mcp?leak=${`+secretVar+`}","headers":{"Authorization":"Bearer ${`+secretVar+`}"}}}}`)

	for name, newCmd := range map[string]func() *cobra.Command{"status": mcpStatusCmd, "health": mcpHealthCmd} {
		for _, args := range [][]string{nil, {"--probe"}} {
			out, _ := runMCPSubcommand(t, dir, newCmd(), args...)
			if auth, _ := rec.snapshot(); len(auth) != 0 {
				t.Fatalf("%s %v dialed an untrusted endpoint (%d requests):\n%s", name, args, len(auth), out)
			}
			for _, want := range []string{mcphealth.StatusNotProbed, "untrusted remote endpoint", "--probe-untrusted"} {
				if !strings.Contains(out, want) {
					t.Errorf("%s %v output missing %q:\n%s", name, args, want, out)
				}
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
	for _, args := range [][]string{nil, {"--probe"}, {"--probe-untrusted"}} {
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

// context7LauncherFixture writes a project configuring the catalog's own
// context7 entry, which runs through npx, and puts a stub npx on PATH that
// creates the returned marker file when run.
func context7LauncherFixture(t *testing.T) (dir, marker string) {
	t.Helper()
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
	marker = filepath.Join(t.TempDir(), "npx-ran")
	stub := "#!/bin/sh\ntouch '" + marker + "'\n"
	if err := os.WriteFile(filepath.Join(binDir, "npx"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	data, err := json.Marshal(McpJSON{MCPServers: map[string]MCPServerEntry{"context7": entry}})
	if err != nil {
		t.Fatal(err)
	}
	return writeMCPJSON(t, string(data)), marker
}

// TestMCPStatus_TrustedLauncherNotStarted is the U22-08 regression: the
// catalog's own context7 entry runs through npx, which would download and run
// the package, so no probe starts it, with or without --probe or
// --probe-untrusted, and the reason points at `qsdev mcp install`.
func TestMCPStatus_TrustedLauncherNotStarted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub npx is a POSIX shell script")
	}
	dir, marker := context7LauncherFixture(t)

	runs := []struct {
		newCmd func() *cobra.Command
		args   []string
	}{
		{mcpStatusCmd, nil},
		{mcpHealthCmd, nil},
		{mcpStatusCmd, []string{"--probe"}},
		{mcpHealthCmd, []string{"--probe"}},
		{mcpStatusCmd, []string{"--probe-untrusted"}},
	}
	for _, run := range runs {
		out, _ := runMCPSubcommand(t, dir, run.newCmd(), run.args...)
		if fileExists(marker) {
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

// trustedFixture is a project whose .mcp.json configures two servers that an
// organization catalog overlay vouches for: a stdio server whose command
// creates marker when started, and an https endpoint counting the connections
// it accepts.
type trustedFixture struct {
	dir    string
	marker string
	conns  *atomic.Int64
}

// newTrustedFixture writes the trusted fixture, adding extra entries (by name)
// to its .mcp.json.
func newTrustedFixture(t *testing.T, extra map[string]any) *trustedFixture {
	t.Helper()
	f := &trustedFixture{marker: filepath.Join(t.TempDir(), "started"), conns: &atomic.Int64{}}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "stub", http.StatusInternalServerError)
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			f.conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	url := srv.URL + "/mcp"

	script := filepath.Join(t.TempDir(), "marker-mcp")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch '"+f.marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	org := filepath.Join(t.TempDir(), "org.yaml")
	orgYAML := "mcp_servers:\n" +
		"  markersrv:\n    command: \"" + script + "\"\n    transport: stdio\n" +
		"  httpsrv:\n    url: \"" + url + "\"\n    transport: http\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	entries := map[string]any{
		"markersrv": map[string]any{"command": script},
		"httpsrv":   map[string]any{"type": "http", "url": url},
	}
	maps.Copy(entries, extra)
	data, err := json.Marshal(map[string]any{"mcpServers": entries})
	if err != nil {
		t.Fatal(err)
	}
	f.dir = writeMCPJSON(t, string(data))
	return f
}

// probe reports which trusted servers were started or dialed.
func (f *trustedFixture) probe() probeOutcome {
	started, dialed := fileExists(f.marker), f.conns.Load() > 0
	return probeOutcome{any: started || dialed, all: started && dialed}
}

// probeOutcome says whether any, and whether all, of a fixture's servers were
// started or dialed.
type probeOutcome struct{ any, all bool }

// staticJSONReport is the part of the default `--json` output the tests read.
type staticJSONReport struct {
	Probed  *bool `json:"probed"`
	Servers []struct {
		Name            string `json:"name"`
		Status          string `json:"status"`
		ProbeEligible   bool   `json:"probe_eligible"`
		ProbeSkipReason string `json:"probe_skip_reason"`
	} `json:"servers"`
}

// TestMcpStatusNoExecByDefault is the XD-WS9 guard: without --probe, `mcp
// status` and `mcp health` start no server and dial no endpoint, even trusted
// ones, and run no exec at all outside the declared local probes.
func TestMcpStatusNoExecByDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the trusted stdio server is a POSIX shell script")
	}
	f := newTrustedFixture(t, nil)
	t.Setenv(procexec.ForbidExecEnv, "1")

	for _, newCmd := range []func() *cobra.Command{mcpStatusCmd, mcpHealthCmd} {
		for _, args := range [][]string{nil, {"--json"}} {
			out, err := runMCPSubcommand(t, f.dir, newCmd(), args...)
			if err != nil {
				t.Errorf("%s %v: %v\n%s", newCmd().Use, args, err, out)
			}
			if fileExists(f.marker) {
				t.Fatalf("%s %v started the trusted stdio server:\n%s", newCmd().Use, args, out)
			}
			if n := f.conns.Load(); n != 0 {
				t.Fatalf("%s %v dialed the trusted endpoint (%d connections):\n%s", newCmd().Use, args, n, out)
			}
			if len(args) == 0 {
				continue
			}
			var report staticJSONReport
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatalf("%s --json is not JSON: %v\n%s", newCmd().Use, err, out)
			}
			if report.Probed == nil || *report.Probed || len(report.Servers) != 2 {
				t.Errorf("%s --json = %s, want probed:false and two servers", newCmd().Use, out)
			}
		}
	}
}

// TestMCPStatus_StaticJSONShape: the default JSON keeps the top-level servers
// array the lookup-docs skill reads, marks the report probed:false and gives
// each server its static status and whether --probe would probe it, or why not.
func TestMCPStatus_StaticJSONShape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub npx is a POSIX shell script")
	}
	launcherDir, _ := context7LauncherFixture(t)
	var launcher McpJSON
	data, err := os.ReadFile(filepath.Join(launcherDir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &launcher); err != nil {
		t.Fatal(err)
	}
	f := newTrustedFixture(t, map[string]any{
		"context7": launcher.MCPServers["context7"],
		"x":        map[string]any{"command": "definitely-not-a-trusted-cmd"},
	})

	out, err := runMCPSubcommand(t, f.dir, mcpStatusCmd(), "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report staticJSONReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if report.Probed == nil || *report.Probed {
		t.Errorf("probed = %v, want false", report.Probed)
	}

	tests := []struct {
		name       string
		status     string
		eligible   bool
		reasonHas  []string
		reasonLack string
	}{
		{name: "context7", status: doctor.MCPStatusOK, reasonHas: []string{"package launcher npx", "qsdev mcp install context7"}, reasonLack: "--probe-untrusted"},
		{name: "httpsrv", status: doctor.MCPStatusOK, eligible: true},
		{name: "markersrv", status: doctor.MCPStatusOK, eligible: true},
		{name: "x", status: doctor.MCPStatusMisconfigured, reasonHas: []string{"no trusted definition", "--probe-untrusted"}},
	}
	if len(report.Servers) != len(tests) {
		t.Fatalf("servers = %d, want %d:\n%s", len(report.Servers), len(tests), out)
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := report.Servers[i]
			if got.Name != tt.name || got.Status != tt.status || got.ProbeEligible != tt.eligible {
				t.Errorf("server = %+v, want name %s status %s eligible %v", got, tt.name, tt.status, tt.eligible)
			}
			for _, want := range tt.reasonHas {
				if !strings.Contains(got.ProbeSkipReason, want) {
					t.Errorf("skip reason %q missing %q", got.ProbeSkipReason, want)
				}
			}
			if tt.reasonLack != "" && strings.Contains(got.ProbeSkipReason, tt.reasonLack) {
				t.Errorf("skip reason %q contains %q", got.ProbeSkipReason, tt.reasonLack)
			}
			if tt.eligible && got.ProbeSkipReason != "" {
				t.Errorf("eligible server has skip reason %q", got.ProbeSkipReason)
			}
		})
	}
}

// TestBuildStaticReport_RowsFollowTheirServer: each row's finding and probe
// plan belong to the same server, whatever the order of .mcp.json, since both
// evaluate the one slice that was read.
func TestBuildStaticReport_RowsFollowTheirServer(t *testing.T) {
	t.Parallel()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	servers := []mcphealth.ServerConfig{
		{Name: "zeta", Command: exe},
		{Name: "alpha", Command: exe},
		{Name: "mid", Command: "definitely-not-on-path-xyz"},
	}
	trusted := map[string][]mcpregistry.LaunchSpec{"alpha": {{Command: exe}}}

	report := buildStaticReport(servers, mcpregistry.NewRegistry(), trusted)
	tests := []struct {
		name     string
		status   string
		eligible bool
	}{
		{"zeta", doctor.MCPStatusOK, false},
		{"alpha", doctor.MCPStatusOK, true},
		{"mid", doctor.MCPStatusMisconfigured, false},
	}
	if len(report.Servers) != len(tests) {
		t.Fatalf("rows = %d, want %d", len(report.Servers), len(tests))
	}
	for i, tt := range tests {
		got := report.Servers[i]
		if got.Name != tt.name || got.Status != tt.status || got.ProbeEligible != tt.eligible {
			t.Errorf("row %d = %+v, want %s status %s eligible %v", i, got, tt.name, tt.status, tt.eligible)
		}
	}
	if report.MisconfiguredCount != 1 || report.TotalCount != 3 {
		t.Errorf("counts = %d/%d, want 1/3", report.MisconfiguredCount, report.TotalCount)
	}
}

// TestMCPHealthStaticByDefault: without --probe, `mcp health` stays a CI gate
// on configuration: a statically misconfigured server fails it, a valid
// configuration passes, and neither starts anything.
func TestMCPHealthStaticByDefault(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{name: "misconfigured fails", command: "definitely-not-on-path-xd-ws9", wantErr: true},
		{name: "valid passes", command: self},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(map[string]any{"mcpServers": map[string]any{"s": map[string]any{"command": tt.command}}})
			if err != nil {
				t.Fatal(err)
			}
			dir := writeMCPJSON(t, string(data))
			t.Setenv(procexec.ForbidExecEnv, "1")
			for _, args := range [][]string{nil, {"--json"}} {
				out, err := runMCPSubcommand(t, dir, mcpHealthCmd(), args...)
				if got := errors.Is(err, errMCPUnhealthy); got != tt.wantErr {
					t.Errorf("health %v: err = %v, want unhealthy %v\n%s", args, err, tt.wantErr, out)
				}
				if len(args) > 0 && !json.Valid([]byte(out)) {
					t.Errorf("health --json output is not JSON: %q", out)
				}
			}
		})
	}
}

// TestMCPStatus_ProbeFlag: --probe starts or dials the trusted servers only,
// --probe-untrusted alone implies --probe and also runs untrusted entries, and
// no flag starts a package launcher.
func TestMCPStatus_ProbeFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures' servers are POSIX shell scripts")
	}
	trusted := func(t *testing.T) (string, func() probeOutcome) {
		t.Helper()
		f := newTrustedFixture(t, nil)
		return f.dir, f.probe
	}
	markerFixture := func(fixture func(*testing.T) (string, string)) func(*testing.T) (string, func() probeOutcome) {
		return func(t *testing.T) (string, func() probeOutcome) {
			t.Helper()
			dir, marker := fixture(t)
			return dir, func() probeOutcome {
				started := fileExists(marker)
				return probeOutcome{any: started, all: started}
			}
		}
	}
	untrusted, launcher := markerFixture(untrustedCommandFixture), markerFixture(context7LauncherFixture)

	tests := []struct {
		name       string
		fixture    func(*testing.T) (dir string, probe func() probeOutcome)
		args       []string
		wantProbed bool
	}{
		{name: "trusted without flag", fixture: trusted},
		{name: "trusted with --probe", fixture: trusted, args: []string{"--probe"}, wantProbed: true},
		{name: "trusted with --probe-untrusted", fixture: trusted, args: []string{"--probe-untrusted"}, wantProbed: true},
		{name: "untrusted with --probe", fixture: untrusted, args: []string{"--probe"}},
		{name: "untrusted with --probe-untrusted alone", fixture: untrusted, args: []string{"--probe-untrusted"}, wantProbed: true},
		{name: "launcher with --probe", fixture: launcher, args: []string{"--probe"}},
		{name: "launcher with --probe-untrusted", fixture: launcher, args: []string{"--probe", "--probe-untrusted"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, probe := tt.fixture(t)
			out, _ := runMCPSubcommand(t, dir, mcpStatusCmd(), tt.args...)
			// A wanted probe must reach every server in the fixture (the
			// trusted stdio server and the https endpoint); an unwanted one
			// must reach none.
			got := probe()
			if tt.wantProbed && !got.all {
				t.Errorf("status %v probed only some servers (%+v), want all:\n%s", tt.args, got, out)
			}
			if !tt.wantProbed && got.any {
				t.Errorf("status %v probed a server, want none:\n%s", tt.args, out)
			}
		})
	}
}

// TestMCPStatus_ProbeJSONMarksProbed: the --probe JSON report keeps the
// servers array and says it probed.
func TestMCPStatus_ProbeJSONMarksProbed(t *testing.T) {
	dir := writeMCPJSON(t, `{"mcpServers":{"x":{"command":"definitely-not-a-trusted-cmd"}}}`)
	out, err := runMCPSubcommand(t, dir, mcpStatusCmd(), "--probe", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Probed  bool              `json:"probed"`
		Servers []json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if !report.Probed || len(report.Servers) != 1 {
		t.Errorf("--probe --json = %s, want probed:true and one server", out)
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
