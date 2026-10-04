package projectctx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// stubMarkerOnPath puts a stub npx on PATH that creates a marker file when it
// runs, and returns the marker path. It skips on Windows, where a POSIX shell
// script is not runnable.
func stubMarkerOnPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub launcher is a POSIX shell script")
	}
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "npx-ran")
	script := "#!/bin/sh\ntouch '" + marker + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "npx"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing stub npx: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return marker
}

// assertNoFile fails when path exists.
func assertNoFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("%s exists: a probe started a server it must not start", path)
	}
}

// listHealth calls qsdev_mcp_list with health=true and returns each server's
// health payload by name.
func listHealth(t *testing.T, pc *ProjectContext) map[string]map[string]any {
	t.Helper()
	res := callTool(t, pc, toolMCPList, map[string]any{"health": true})
	if res.IsError {
		t.Fatalf("mcp_list error: %q", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if structured["health_probed"] != true {
		t.Errorf("health_probed = %v, want true", structured["health_probed"])
	}
	out := make(map[string]map[string]any)
	for _, s := range structured["servers"].([]map[string]any) {
		h, ok := s["health"].(map[string]any)
		if !ok {
			t.Fatalf("server %v has no health payload", s["name"])
		}
		out[s["name"].(string)] = h
	}
	return out
}

// TestMCPList_HealthConfiguredOnly: health covers only the servers the
// project's .mcp.json configures; every other registry entry is reported as
// not configured and nothing is started.
func TestMCPList_HealthConfiguredOnly(t *testing.T) {
	marker := stubMarkerOnPath(t)
	dir, pc := newGoProject(t)
	writeFile(t, dir, ".mcp.json", `{"mcpServers": {}}`)

	health := listHealth(t, pc)
	if len(health) == 0 {
		t.Fatal("mcp_list returned no servers")
	}
	for name, h := range health {
		if h["status"] != healthNotConfigured {
			t.Errorf("%s health = %v, want status %q", name, h, healthNotConfigured)
		}
	}
	assertNoFile(t, marker)
}

// TestMCPList_HealthUntrustedRedefinitionNotProbed: a .mcp.json entry that
// reuses a catalog name with a different command is untrusted, so it is not
// probed and its command never runs; mcp.list never offers the override.
func TestMCPList_HealthUntrustedRedefinitionNotProbed(t *testing.T) {
	marker := stubMarkerOnPath(t)
	dir, pc := newGoProject(t)
	writeFile(t, dir, ".mcp.json", `{"mcpServers": {"context7": {"command": "/bin/sh", "args": ["-c", "touch '`+marker+`'"]}}}`)

	h := listHealth(t, pc)["context7"]
	if h["status"] != mcphealth.StatusNotProbed {
		t.Errorf("context7 health = %v, want status %q", h, mcphealth.StatusNotProbed)
	}
	if reason, _ := h["error"].(string); !strings.Contains(reason, "no trusted definition") {
		t.Errorf("context7 reason = %q, want it to name the missing trusted definition", reason)
	}
	assertNoFile(t, marker)
}

// TestMCPList_HealthTrustedLauncherNotStarted: a configured entry that exactly
// matches its catalog definition is still not started when that definition is
// a package launcher (it would download and run the package).
func TestMCPList_HealthTrustedLauncherNotStarted(t *testing.T) {
	marker := stubMarkerOnPath(t)
	dir, pc := newGoProject(t)
	writeFile(t, dir, ".mcp.json", `{"mcpServers": {"context7": {"command": "npx", "args": ["@upstash/context7-mcp@4.1.1"]}}}`)

	h := listHealth(t, pc)["context7"]
	if h["status"] != mcphealth.StatusNotProbed {
		t.Errorf("context7 health = %v, want status %q", h, mcphealth.StatusNotProbed)
	}
	if reason, _ := h["error"].(string); !strings.Contains(reason, "mcp install context7") {
		t.Errorf("context7 reason = %q, want it to point at mcp install", reason)
	}
	assertNoFile(t, marker)
}

// TestMCPList_HealthTrustedConfiguredProbed: a configured entry matching its
// trusted definition (here an organization overlay pointing the catalog's
// http server at a loopback listener) is actually probed.
func TestMCPList_HealthTrustedConfiguredProbed(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "stub", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	org := filepath.Join(t.TempDir(), "org.yaml")
	orgYAML := "mcp_servers:\n  socket:\n    url: \"" + srv.URL + "/mcp\"\n    transport: http\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatalf("writing org overlay: %v", err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	dir, pc := newGoProject(t)
	writeFile(t, dir, ".mcp.json", `{"mcpServers": {"socket": {"type": "http", "url": "`+srv.URL+`/mcp"}}}`)

	h := listHealth(t, pc)["socket"]
	if h["status"] == mcphealth.StatusNotProbed || h["status"] == healthNotConfigured {
		t.Errorf("socket health = %v, want a live probe result", h)
	}
	if hits.Load() == 0 {
		t.Error("the trusted, configured endpoint received no probe request")
	}
}

// TestMCPList_HealthBinaryConfiguredTrusted: a server definition configured
// into the binary (WithTrustedServers, as `qsdev mcp serve` passes it) is
// trusted exactly as `qsdev mcp status` trusts it; without it the same entry
// is an untrusted redefinition and is not probed.
func TestMCPList_HealthBinaryConfiguredTrusted(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "missing-mcp-server")
	spec := mcpregistry.LaunchSpec{Command: bin, Args: []string{"--stdio"}}
	tests := []struct {
		name      string
		opts      []Option
		wantProbe bool
	}{
		{name: "binary-configured definition trusted", opts: []Option{WithTrustedServers(map[string][]mcpregistry.LaunchSpec{"context7": {spec}})}, wantProbe: true},
		{name: "without it untrusted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, pc := newGoProject(t)
			for _, o := range tt.opts {
				o(pc)
			}
			mcpJSON, err := json.Marshal(map[string]any{"mcpServers": map[string]any{"context7": map[string]any{"command": spec.Command, "args": spec.Args}}})
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, dir, ".mcp.json", string(mcpJSON))

			h := listHealth(t, pc)["context7"]
			if probed := h["status"] != mcphealth.StatusNotProbed; probed != tt.wantProbe {
				t.Errorf("context7 health = %v, probed = %v, want %v", h, probed, tt.wantProbe)
			}
		})
	}
}
