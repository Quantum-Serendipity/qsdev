package mcpregistry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// TestProbeSkipReason covers the launch-safety classification health probes use
// to avoid downloading and running packages or spawning qsdev's own server.
func TestProbeSkipReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      mcphealth.ServerConfig
		wantSkip bool
	}{
		{"npx launcher", mcphealth.ServerConfig{Command: "npx", Args: []string{"-y", "pkg"}}, true},
		{"absolute uvx", mcphealth.ServerConfig{Command: "/nix/store/x-uv/bin/uvx", Args: []string{"pkg"}}, true},
		{"pipx", mcphealth.ServerConfig{Command: "pipx", Args: []string{"run", "pkg"}}, true},
		{"windows npx shim", mcphealth.ServerConfig{Command: `C:\Program Files\nodejs\npx.cmd`, Args: []string{"-y", "pkg"}}, true},
		{"cmd /c wrapper", mcphealth.ServerConfig{Command: "cmd", Args: []string{"/c", "npx", "-y", "pkg"}}, true},
		{"sh -c wrapper", mcphealth.ServerConfig{Command: "sh", Args: []string{"-c", "uvx --from semble[mcp] semble"}}, true},
		{"env wrapper", mcphealth.ServerConfig{Command: "/usr/bin/env", Args: []string{"FOO=1", "bunx", "pkg"}}, true},
		{"self server", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--stdio"}}, true},
		{"self server absolute", mcphealth.ServerConfig{Command: "/run/current-system/sw/bin/qsdev", Args: []string{"mcp", "serve"}}, true},
		{"empty command", mcphealth.ServerConfig{}, true},
		{"other qsdev subcommand", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "health"}}, false},
		{"single-module server", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--module", "agent-postmortem"}}, false},
		{"single-module server, joined flag", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--module=version-sentinel"}}, false},
		{"local binary", mcphealth.ServerConfig{Command: "/opt/bin/server", Args: []string{"--port", "0"}}, false},
		{"launcher name as substring", mcphealth.ServerConfig{Command: "/opt/bin/npx-free-server"}, false},
		{"remote url", mcphealth.ServerConfig{URL: "https://example.test/mcp"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := probeSkipReason(tt.cfg) != ""; got != tt.wantSkip {
				t.Errorf("probeSkipReason(%+v) skip = %v, want %v", tt.cfg, got, tt.wantSkip)
			}
		})
	}
}

// TestPlanProbes covers the single live-probe gate (U22-08, U22-02, U21-V01,
// F095): the launcher/self rule and the https-or-loopback rule always apply,
// and an entry matching no trusted definition is probed only on request, with
// its remote env references left unexpanded.
func TestPlanProbes(t *testing.T) {
	t.Parallel()

	trusted := testTrusted()
	trusted["svc"] = []LaunchSpec{{Command: "/opt/bin/svc", Args: []string{"--stdio"}}}
	trusted["wrapped"] = []LaunchSpec{{Command: "sh", Args: []string{"-c", "uvx x"}}}
	trusted["qsdev"] = []LaunchSpec{{Command: "qsdev", Args: []string{"mcp", "serve"}}}
	trusted["local"] = []LaunchSpec{{URL: "http://127.0.0.1:9/mcp"}}

	tests := []struct {
		name            string
		cfg             mcphealth.ServerConfig
		allowUntrusted  bool
		wantProbe       bool
		wantExpand      bool
		wantReason      []string
		wantNoReason    []string
		wantOverridable bool
	}{
		{name: "untrusted stdio", cfg: mcphealth.ServerConfig{Name: "x", Command: "/opt/bin/evil"}, wantReason: []string{"no trusted definition", `"/opt/bin/evil"`}, wantOverridable: true},
		{name: "known name, different args", cfg: mcphealth.ServerConfig{Name: "svc", Command: "/opt/bin/svc", Args: []string{"--evil"}}, wantReason: []string{"no trusted definition"}, wantOverridable: true},
		{name: "known name, injected env", cfg: mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"NODE_OPTIONS": "--require=/tmp/x.js"}}, wantReason: []string{"no trusted definition"}, wantOverridable: true},
		{name: "trusted stdio", cfg: mcphealth.ServerConfig{Name: "svc", Command: "/opt/bin/svc", Args: []string{"--stdio"}}, wantProbe: true, wantExpand: true},
		{name: "trusted stdio with env", cfg: mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}, wantProbe: true, wantExpand: true},
		{name: "untrusted stdio allowed", cfg: mcphealth.ServerConfig{Name: "x", Command: "/opt/bin/evil"}, allowUntrusted: true, wantProbe: true},
		{name: "trusted npx launcher", cfg: mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}, wantReason: []string{"package launcher npx", "qsdev mcp install context7"}},
		{name: "trusted npx launcher allowed", cfg: mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}, allowUntrusted: true, wantReason: []string{"package launcher npx", "qsdev mcp install context7"}},
		{name: "untrusted npx launcher allowed", cfg: mcphealth.ServerConfig{Name: "x", Command: "npx", Args: []string{"-y", "evil"}}, allowUntrusted: true, wantReason: []string{"package launcher npx", "install a pinned release locally"}, wantNoReason: []string{"mcp install"}},
		{name: "catalog launcher without install method", cfg: mcphealth.ServerConfig{Name: "filesystem", Command: "npx", Args: []string{"@modelcontextprotocol/server-filesystem@2026.8.31", "."}}, allowUntrusted: true, wantReason: []string{"package launcher npx", "install a pinned release locally"}, wantNoReason: []string{"mcp install"}},
		{name: "env-wrapped launcher", cfg: mcphealth.ServerConfig{Name: "x", Command: "env", Args: []string{"-i", "NPX", "x"}}, allowUntrusted: true, wantReason: []string{"package launcher npx"}, wantNoReason: []string{"launcher env"}},
		{name: "env path with assignment", cfg: mcphealth.ServerConfig{Name: "x", Command: "/usr/bin/env", Args: []string{"FOO=1", "npx", "x"}}, allowUntrusted: true, wantReason: []string{"package launcher npx"}, wantNoReason: []string{"launcher env"}},
		{name: "sh -c uvx wrapper", cfg: mcphealth.ServerConfig{Name: "wrapped", Command: "sh", Args: []string{"-c", "uvx x"}}, allowUntrusted: true, wantReason: []string{"package launcher uvx"}},
		{name: "self server", cfg: mcphealth.ServerConfig{Name: "qsdev", Command: "qsdev", Args: []string{"mcp", "serve"}}, allowUntrusted: true, wantReason: []string{"this qsdev MCP server"}},
		{name: "untrusted url", cfg: mcphealth.ServerConfig{Name: "h", URL: "https://evil.invalid/mcp?leak=${U21_T}", Headers: map[string]string{"Authorization": "Bearer ${U21_T}"}}, wantReason: []string{"untrusted remote endpoint", "no trusted definition"}, wantOverridable: true},
		{name: "trusted url", cfg: mcphealth.ServerConfig{Name: "remote", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"}}, wantProbe: true, wantExpand: true},
		{name: "trusted url, changed header template", cfg: mcphealth.ServerConfig{Name: "remote", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${GITHUB_TOKEN}"}}, wantReason: []string{"untrusted remote endpoint"}, wantOverridable: true},
		{name: "trusted url under another name", cfg: mcphealth.ServerConfig{Name: "context7", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"}}, wantReason: []string{"untrusted remote endpoint"}, wantOverridable: true},
		{name: "plain http remote allowed", cfg: mcphealth.ServerConfig{Name: "h", URL: "http://example.com/mcp"}, allowUntrusted: true, wantReason: []string{"plain http to a non-local host"}},
		{name: "plain http remote not allowed", cfg: mcphealth.ServerConfig{Name: "h", URL: "http://example.com/mcp"}, wantReason: []string{"plain http to a non-local host"}},
		{name: "plain http loopback allowed", cfg: mcphealth.ServerConfig{Name: "h", URL: "http://127.0.0.1:9/mcp", Headers: map[string]string{"Authorization": "Bearer ${U21_T}"}}, allowUntrusted: true, wantProbe: true},
		{name: "plain http loopback trusted", cfg: mcphealth.ServerConfig{Name: "local", URL: "http://127.0.0.1:9/mcp"}, wantProbe: true, wantExpand: true},
		{name: "untrusted entry cannot request expansion", cfg: mcphealth.ServerConfig{Name: "h", URL: "http://127.0.0.1:9/mcp", ExpandEnv: true}, allowUntrusted: true, wantProbe: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := []mcphealth.ServerConfig{tc.cfg}
			probe, skipped := PlanProbes(in, trusted, tc.allowUntrusted)
			if in[0].ExpandEnv != tc.cfg.ExpandEnv {
				t.Error("PlanProbes mutated its input")
			}
			if tc.wantProbe {
				if len(probe) != 1 || len(skipped) != 0 {
					t.Fatalf("probe = %+v, skipped = %+v, want it probed", probe, skipped)
				}
				if probe[0].ExpandEnv != tc.wantExpand {
					t.Errorf("ExpandEnv = %v, want %v", probe[0].ExpandEnv, tc.wantExpand)
				}
				return
			}
			if len(probe) != 0 || len(skipped) != 1 {
				t.Fatalf("probe = %+v, skipped = %+v, want it skipped", probe, skipped)
			}
			s := skipped[0]
			if s.Name != tc.cfg.Name {
				t.Errorf("skip name = %q, want %q", s.Name, tc.cfg.Name)
			}
			for _, want := range tc.wantReason {
				if !strings.Contains(s.Reason, want) {
					t.Errorf("reason = %q, want it to contain %q", s.Reason, want)
				}
			}
			for _, unwanted := range tc.wantNoReason {
				if strings.Contains(s.Reason, unwanted) {
					t.Errorf("reason = %q, want it not to contain %q", s.Reason, unwanted)
				}
			}
			if s.Overridable != tc.wantOverridable {
				t.Errorf("Overridable = %v, want %v (reason %q)", s.Overridable, tc.wantOverridable, s.Reason)
			}
		})
	}
}

// TestProbeAll_NotProbedEntriesSorted checks that ProbeAll reports every
// server once, sorted by name, with skipped servers as not-probed carrying
// their reason (plus the override hint only where the override would help),
// and that only the planned servers are contacted.
func TestProbeAll_NotProbedEntriesSorted(t *testing.T) {
	t.Parallel()

	var (
		mu   sync.Mutex
		hits = map[string]int{}
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		http.Error(w, "no", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	trusted := map[string][]LaunchSpec{"b-remote": {{URL: srv.URL}}}
	servers := []mcphealth.ServerConfig{
		{Name: "d-untrusted", Command: "/opt/bin/evil"},
		{Name: "b-remote", URL: srv.URL},
		{Name: "a-launcher", Command: "npx", Args: []string{"-y", "pkg"}},
		{Name: "c-untrusted-url", URL: srv.URL + "/other"},
	}
	const hint = "; rerun with --probe-untrusted"
	report := ProbeAll(context.Background(), servers, trusted, ProbeOptions{Timeout: 5 * time.Second, OverrideHint: hint})

	if report.TotalCount != len(servers) || len(report.Servers) != len(servers) {
		t.Fatalf("total = %d, servers = %+v, want %d entries", report.TotalCount, report.Servers, len(servers))
	}
	names := make([]string, 0, len(report.Servers))
	byName := map[string]mcphealth.ServerHealth{}
	for _, s := range report.Servers {
		names = append(names, s.Name)
		byName[s.Name] = s
	}
	if !slices.IsSorted(names) {
		t.Errorf("servers not sorted: %v", names)
	}
	if got := byName["b-remote"].Status; got != mcphealth.StatusUnreachable {
		t.Errorf("b-remote status = %q, want %q (probed)", got, mcphealth.StatusUnreachable)
	}
	for _, name := range []string{"a-launcher", "c-untrusted-url", "d-untrusted"} {
		if got := byName[name].Status; got != mcphealth.StatusNotProbed {
			t.Errorf("%s status = %q, want %q", name, got, mcphealth.StatusNotProbed)
		}
	}
	if strings.Contains(byName["a-launcher"].Error, hint) {
		t.Errorf("launcher skip suggests an override that cannot help: %q", byName["a-launcher"].Error)
	}
	for _, name := range []string{"c-untrusted-url", "d-untrusted"} {
		if !strings.HasSuffix(byName[name].Error, hint) {
			t.Errorf("%s error = %q, want the override hint", name, byName[name].Error)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if hits["/"] == 0 {
		t.Error("trusted endpoint was not probed")
	}
	if hits["/other"] != 0 {
		t.Errorf("untrusted endpoint was contacted %d times", hits["/other"])
	}
}

// TestPlanProbes_JudgesExpandedTarget checks that the rules no override lifts
// see what the probe would actually start or dial, so an environment value
// cannot turn a harmless-looking template into a launcher or a plain-http
// remote endpoint.
func TestPlanProbes_JudgesExpandedTarget(t *testing.T) {
	t.Setenv("U22_LAUNCHER", "npx")
	t.Setenv("U22_HOST", "example.com")
	trusted := map[string][]LaunchSpec{"remote": {{URL: "http://${U22_HOST}/mcp"}}}
	servers := []mcphealth.ServerConfig{
		{Name: "cmd", Command: "${U22_LAUNCHER}", Args: []string{"-y", "pkg"}},
		{Name: "remote", URL: "http://${U22_HOST}/mcp"},
	}
	probe, skipped := PlanProbes(servers, trusted, true)
	if len(probe) != 0 {
		t.Fatalf("probe = %+v, want none", probe)
	}
	want := map[string]string{"cmd": "package launcher npx", "remote": "plain http to a non-local host"}
	for _, s := range skipped {
		if !strings.Contains(s.Reason, want[s.Name]) || s.Overridable {
			t.Errorf("%s: reason = %q, overridable = %v; want %q, not overridable", s.Name, s.Reason, s.Overridable, want[s.Name])
		}
	}
}
