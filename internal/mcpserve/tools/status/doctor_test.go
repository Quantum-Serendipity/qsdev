package status

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCheckMCPProbesOnlySafeConfiguredServers is the regression test for the
// doctor launching the whole built-in catalog: it must probe only the servers the
// project's .mcp.json configures, and never start a package launcher (which
// would download and run an unpinned package) or qsdev's own MCP server.
func TestCheckMCPProbesOnlySafeConfiguredServers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mcpJSON := `{"mcpServers": {
  "context7": {"command": "npx", "args": ["-y", "@upstash/context7-mcp"]},
  "semble":   {"command": "/usr/bin/uvx", "args": ["--from", "semble[mcp]", "semble"]},
  "qsdev":    {"command": "qsdev", "args": ["mcp", "serve"]},
  "postmortem": {"command": "qsdev", "args": ["mcp", "agent-postmortem"]},
  "local":    {"command": "/opt/local/bin/local-mcp"},
  "remote":   {"type": "http", "url": "https://mcp.example.test/mcp"}
}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := newDoctorChecker(dir)
	var mu sync.Mutex
	var probed []string
	doc.probeMCP = func(_ context.Context, cfg mcphealth.ServerConfig) *mcphealth.ServerHealth {
		mu.Lock()
		probed = append(probed, cfg.Name)
		mu.Unlock()
		return &mcphealth.ServerHealth{Status: mcphealth.StatusHealthy}
	}

	res := doc.checkMCP(context.Background())
	slices.Sort(probed)
	if want := []string{"local", "postmortem", "remote"}; !slices.Equal(probed, want) {
		t.Errorf("probed %v, want %v", probed, want)
	}
	if res.Status != checkPass {
		t.Errorf("status = %q, want %q", res.Status, checkPass)
	}
	for _, want := range []string{"3 healthy, 0 unhealthy of 3 probed", "3 not probed", "context7", "semble", "qsdev (this qsdev MCP server)"} {
		if !strings.Contains(res.Detail, want) {
			t.Errorf("detail = %q, want it to mention %q", res.Detail, want)
		}
	}
}

// TestCheckMCPNoMcpJSON verifies a project without .mcp.json probes nothing,
// rather than falling back to the global catalog.
func TestCheckMCPNoMcpJSON(t *testing.T) {
	t.Parallel()
	doc := newDoctorChecker(t.TempDir())
	doc.probeMCP = func(_ context.Context, cfg mcphealth.ServerConfig) *mcphealth.ServerHealth {
		t.Errorf("unexpected probe of %q", cfg.Name)
		return &mcphealth.ServerHealth{Status: mcphealth.StatusHealthy}
	}
	if res := doc.checkMCP(context.Background()); res.Status != checkPass || !strings.Contains(res.Detail, "no MCP servers configured") {
		t.Errorf("got %+v, want pass with no configured servers", res)
	}
}

// TestCheckMCPInvalidMcpJSON verifies a malformed .mcp.json fails the check.
func TestCheckMCPInvalidMcpJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := newDoctorChecker(dir).checkMCP(context.Background()); res.Status != checkFail {
		t.Errorf("status = %q, want %q for invalid .mcp.json", res.Status, checkFail)
	}
}

// TestDoctorTimeoutBoundsChecksThatIgnoreContext is the regression test for the
// advertised deadline: a check that ignores ctx (e.g. a hung subprocess) must not
// hold the tool call past the doctor's timeout.
func TestDoctorTimeoutBoundsChecksThatIgnoreContext(t *testing.T) {
	t.Parallel()
	doc := newDoctorChecker(t.TempDir())
	doc.timeout = 50 * time.Millisecond

	release := make(chan struct{})
	defer close(release)
	checks := []namedCheck{
		{"fast", func(context.Context) checkResult { return checkResult{"fast", checkPass, "ok", ""} }},
		{"hung", func(context.Context) checkResult {
			<-release // ignores ctx entirely
			return checkResult{"hung", checkPass, "late", ""}
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), doc.timeout)
	defer cancel()
	start := time.Now()
	results := doc.runChecks(ctx, checks)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("runChecks took %s, want it bounded by the %s timeout", elapsed, doc.timeout)
	}
	if results[0].Status != checkPass {
		t.Errorf("fast check = %+v, want pass", results[0])
	}
	if results[1].Name != "hung" || results[1].Status != checkFail || !strings.Contains(results[1].Detail, "timed out") {
		t.Errorf("hung check = %+v, want a timed-out failure", results[1])
	}
}

// TestToolchainBinaries verifies the tools check covers every language the
// shared language detector reports, not just Go/Node/Rust/Python.
func TestToolchainBinaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		det  types.DetectedProject
		want []string
	}{
		{"none", types.DetectedProject{}, nil},
		{"go", types.DetectedProject{HasGoMod: true}, []string{"go"}},
		{"maven", types.DetectedProject{HasPomXML: true}, []string{"java"}},
		{"gradle", types.DetectedProject{HasBuildGradle: true}, []string{"java"}},
		{"maven and gradle", types.DetectedProject{HasPomXML: true, HasBuildGradle: true}, []string{"java"}},
		{"dotnet", types.DetectedProject{HasCsproj: true}, []string{"dotnet"}},
		{"polyglot", types.DetectedProject{HasGoMod: true, HasPackageJSON: true, HasCargoToml: true, HasPyProject: true},
			[]string{"go", "node", "cargo", "python3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := toolchainBinaries(tt.det); !slices.Equal(got, tt.want) {
				t.Errorf("toolchainBinaries() = %v, want %v", got, tt.want)
			}
		})
	}
}
