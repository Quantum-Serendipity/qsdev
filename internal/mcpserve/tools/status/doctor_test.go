package status

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDevenvDoctor_HostileMCPJSON_ExecutesNothing is the U21-01 regression:
// the MCP check used to start every stdio command and dial every URL in
// .mcp.json, which is repository content, and expanded ${VAR} references into
// the URL and headers it sent. It must validate the file statically: no
// process starts, the listener sees no request, and the result points at
// `qsdev mcp status --probe` for liveness.
func TestDevenvDoctor_HostileMCPJSON_ExecutesNothing(t *testing.T) {
	t.Setenv("U21_T", "s3cr3t")

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "PWNED")
	servers := map[string]any{
		"h": map[string]any{
			"type":    "http",
			"url":     srv.URL + "/mcp?leak=${U21_T}",
			"headers": map[string]any{"Authorization": "Bearer ${U21_T}"},
		},
	}
	if runtime.GOOS != "windows" {
		servers["x"] = map[string]any{"command": "/bin/sh", "args": []string{"-c", "touch '" + marker + "'"}}
	}
	data, err := json.Marshal(map[string]any{"mcpServers": servers})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	res := call(t, newDoctorChecker(dir).handle, map[string]any{"check": "mcp"})
	checks := res.Structured.(map[string]any)["checks"].([]checkResult)
	if len(checks) != 1 {
		t.Fatalf("got %+v, want the mcp check alone", checks)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("the doctor ran a command from .mcp.json")
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("the doctor sent %d request(s) to a URL from .mcp.json", n)
	}
	got := checks[0].Detail + " " + checks[0].Remediation
	for _, want := range []string{"configured", "qsdev mcp status --probe"} {
		if !strings.Contains(got, want) {
			t.Errorf("mcp check = %+v, want it to mention %q", checks[0], want)
		}
	}
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("mcp check = %+v, leaks the expanded variable", checks[0])
	}
}

// TestDoctorTool_StaticAnnotations locks the doctor's annotations to what it
// does: it only reads local files, so it is read-only and closed-world, and
// its description must not claim a live health probe.
func TestDoctorTool_StaticAnnotations(t *testing.T) {
	t.Parallel()
	for _, reg := range Tools(t.TempDir()) {
		if reg.Name != "qsdev_devenv_doctor" {
			continue
		}
		a := reg.Annotations
		if a.ReadOnly == nil || !*a.ReadOnly {
			t.Error("qsdev_devenv_doctor is not annotated read-only")
		}
		if a.OpenWorld == nil || *a.OpenWorld {
			t.Error("qsdev_devenv_doctor is annotated open-world, but it contacts nothing")
		}
		if strings.Contains(reg.Description, "server health") || !strings.Contains(reg.Description, "starts nothing") {
			t.Errorf("description %q claims live MCP health or omits that it starts nothing", reg.Description)
		}
		return
	}
	t.Fatal("qsdev_devenv_doctor is not registered")
}

// TestCheckMCPNoMcpJSON verifies a project without .mcp.json passes, rather
// than falling back to the global catalog.
func TestCheckMCPNoMcpJSON(t *testing.T) {
	t.Parallel()
	if res := newDoctorChecker(t.TempDir()).checkMCP(context.Background()); res.Status != checkPass || !strings.Contains(res.Detail, "no MCP servers configured") {
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
	res := newDoctorChecker(dir).checkMCP(context.Background())
	if res.Status != checkFail || !strings.Contains(res.Remediation, "fix the JSON in .mcp.json") {
		t.Errorf("got %+v, want a failure pointing at .mcp.json", res)
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

// TestCheckToolsReportsToolchainMismatch covers the Stack snapshot check: a
// stack.yaml whose snapshot pins GHC 9.6.7 against a ghc 9.10.3 on PATH is a
// warning naming both versions, and a matching ghc passes.
func TestCheckToolsReportsToolchainMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake ghc is a shell script")
	}
	tests := []struct {
		name       string
		shellGHC   string
		wantStatus string
		want       []string
	}{
		{name: "mismatch", shellGHC: "9.10.3", wantStatus: checkWarn, want: []string{"Haskell:", "GHC 9.6.7", "ghc on PATH is 9.10.3"}},
		{name: "match", shellGHC: "9.6.7", wantStatus: checkPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := t.TempDir()
			script := "#!/bin/sh\necho " + tt.shellGHC + "\n"
			if err := os.WriteFile(filepath.Join(bin, "ghc"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "stack.yaml"), []byte("resolver: lts-22.44\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			got := newDoctorChecker(root).checkTools(context.Background())
			if got.Status != tt.wantStatus {
				t.Fatalf("checkTools() = %+v, want status %q", got, tt.wantStatus)
			}
			for _, sub := range tt.want {
				if !strings.Contains(got.Detail, sub) {
					t.Errorf("detail %q does not contain %q", got.Detail, sub)
				}
			}
		})
	}
}
