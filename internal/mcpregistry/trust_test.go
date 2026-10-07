package mcpregistry

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// testTrusted is a trusted-definition table with stdio and remote entries.
func testTrusted() map[string][]LaunchSpec {
	return map[string][]LaunchSpec{
		"context7": {{Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}},
		"github":   {{Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}},
		"remote": {{
			URL:     "https://mcp.example.invalid/mcp",
			Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"},
		}},
	}
}

func TestMatchesTrusted(t *testing.T) {
	t.Parallel()

	specs := []LaunchSpec{
		{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}},
		{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}},
	}
	tests := []struct {
		name string
		cfg  mcphealth.ServerConfig
		want bool
	}{
		{"stdio identical", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, true},
		{"stdio different command", mcphealth.ServerConfig{Command: "evil", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio extra arg", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio", "-x"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio different env", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "v"}}, false},
		{"empty command never matches", mcphealth.ServerConfig{}, false},
		{"http identical", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, true},
		{"http different url", mcphealth.ServerConfig{URL: "https://evil.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, false},
		{"http header template changed", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${AWS_SECRET_ACCESS_KEY}"}}, false},
		{"http added env", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}, Env: map[string]string{"A": "b"}}, false},
		{"http url is not a stdio match", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}, URL: "https://evil.invalid/mcp"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesTrusted(tc.cfg, specs); got != tc.want {
				t.Errorf("MatchesTrusted = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLaunchVariants(t *testing.T) {
	t.Parallel()

	env := map[string]string{"K": "${K}"}
	tests := []struct {
		name string
		def  catalog.MCPServerDef
		want []LaunchSpec
	}{
		{
			name: "stdio launcher",
			def:  catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: env, Transport: "stdio"},
			want: []LaunchSpec{
				{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: env},
				{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{
					"K": "${K}", npmIgnoreScriptsEnv: "true", npmBeforeEnv: releaseCutoffPlaceholder,
				}},
			},
		},
		{
			name: "npx allowing install scripts",
			def:  catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, AllowInstallScripts: true},
			want: []LaunchSpec{
				{Command: "npx", Args: []string{"pkg@1.0.0"}},
				{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: releaseCutoffPlaceholder}},
			},
		},
		{
			name: "local command is not hardened",
			def:  catalog.MCPServerDef{Command: "qsdev", Args: []string{"mcp", "serve"}},
			want: []LaunchSpec{{Command: "qsdev", Args: []string{"mcp", "serve"}}},
		},
		{
			name: "http",
			def:  catalog.MCPServerDef{URL: "https://mcp.example.invalid/", Transport: "http", Env: env},
			want: []LaunchSpec{{URL: "https://mcp.example.invalid/", Env: env}},
		},
		{
			name: "installed bin",
			def:  catalog.MCPServerDef{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}, Version: "1.0.0", Bin: "srv", BinArgs: []string{"--db", "x"}},
			want: []LaunchSpec{
				{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}},
				{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv==1.0.0", "--db", "x"}},
				{Command: "srv", Args: []string{"--db", "x"}},
			},
		},
		{
			name: "bin without version is not installable",
			def:  catalog.MCPServerDef{Command: "uvx", Args: []string{"srv"}, Bin: "srv"},
			want: []LaunchSpec{
				{Command: "uvx", Args: []string{"srv"}},
				{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv"}},
			},
		},
		{
			name: "optional args",
			def: catalog.MCPServerDef{
				Command: "uvx", Args: []string{"srv==1.0.0"}, Version: "1.0.0", Bin: "srv",
				OptionalArgs: [][]string{{"--text"}, {"--a", "b"}},
			},
			want: []LaunchSpec{
				{Command: "uvx", Args: []string{"srv==1.0.0"}},
				{Command: "uvx", Args: []string{"srv==1.0.0", "--text"}},
				{Command: "uvx", Args: []string{"srv==1.0.0", "--a", "b"}},
				{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv==1.0.0"}},
				{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv==1.0.0", "--text"}},
				{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv==1.0.0", "--a", "b"}},
				{Command: "srv"},
				{Command: "srv", Args: []string{"--text"}},
				{Command: "srv", Args: []string{"--a", "b"}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := LaunchVariants(tc.def); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("LaunchVariants =\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}

// TestLaunchVariants_DoesNotAliasDef guards against appending optional
// arguments into the catalog definition's own Args backing array.
func TestLaunchVariants_DoesNotAliasDef(t *testing.T) {
	t.Parallel()
	args := make([]string, 1, 8)
	args[0] = "srv==1.0.0"
	def := catalog.MCPServerDef{Command: "uvx", Args: args, OptionalArgs: [][]string{{"--x"}, {"--y"}}}
	got := LaunchVariants(def)
	if want := []string{"srv==1.0.0", "--x"}; !reflect.DeepEqual(got[1].Args, want) {
		t.Errorf("first optional variant = %v, want %v", got[1].Args, want)
	}
}

// TestTrustedDefinitions_ExcludesProjectOverlay: the trusted set is the
// embedded catalog, the user's organization overlay and the binary's own
// servers. The project defaults file is repository content and never
// contributes, even when it is broken.
func TestTrustedDefinitions_ExcludesProjectOverlay(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "org.yaml")
	orgYAML := "mcp_servers:\n  orgsrv:\n    display_name: Org\n    category: agent\n    description: org server\n    command: org-mcp\n    args: [\"--stdio\"]\n    transport: stdio\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	project := filepath.Join(tmp, "project")
	projYAML := "mcp_servers:\n  evil:\n    command: evil\n    transport: stdio\n"
	projFile := catalog.ProjectConfigPath(project)
	if err := os.MkdirAll(filepath.Dir(projFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projFile, []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.ResetDefault()
	t.Cleanup(catalog.ResetDefault)
	if err := catalog.SetProjectRoot(project); err != nil {
		t.Fatalf("SetProjectRoot: %v", err)
	}

	extra := map[string][]LaunchSpec{"bin-srv": {{Command: "bin-srv"}}}
	trusted := TrustedDefinitions(extra)

	if _, ok := trusted["evil"]; ok {
		t.Error("project overlay server is trusted")
	}
	if want := []LaunchSpec{{Command: "org-mcp", Args: []string{"--stdio"}}}; !reflect.DeepEqual(trusted["orgsrv"], want) {
		t.Errorf("org server specs = %#v, want %#v", trusted["orgsrv"], want)
	}
	if !reflect.DeepEqual(trusted["bin-srv"], extra["bin-srv"]) {
		t.Errorf("extra specs = %#v, want %#v", trusted["bin-srv"], extra["bin-srv"])
	}
	embedded, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatal(err)
	}
	for name, def := range embedded.MCPServers() {
		if !reflect.DeepEqual(trusted[name], LaunchVariants(def)) {
			t.Errorf("%s specs = %#v, want LaunchVariants %#v", name, trusted[name], LaunchVariants(def))
		}
	}
}

// TestTrustedDefinitions_BrokenOrgOverlayKeepsEmbedded: TrustedDefinitions
// reads the user scope through catalog.LoadUserScope, so a broken org overlay
// drops only the overlay's servers; the embedded ones stay trusted.
func TestTrustedDefinitions_BrokenOrgOverlayKeepsEmbedded(t *testing.T) {
	org := filepath.Join(t.TempDir(), "org.yaml")
	orgYAML := "mcp_servers:\n  orgsrv:\n    command: org-mcp\n    transport: stdio\nnot_a_section: 1\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	trusted := TrustedDefinitions(nil)
	if _, ok := trusted["orgsrv"]; ok {
		t.Error("server from a broken org overlay is trusted")
	}
	embedded, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatal(err)
	}
	for name, def := range embedded.MCPServers() {
		if !reflect.DeepEqual(trusted[name], LaunchVariants(def)) {
			t.Errorf("%s specs = %#v, want LaunchVariants %#v", name, trusted[name], LaunchVariants(def))
		}
	}
}

// hardenNow is a clock in a non-UTC zone: 2026-09-23T10:30:00Z, so the
// cutoffs below count back from 2026-09-23T00:00:00Z (UTC midnight).
var hardenNow = time.Date(2026, 9, 23, 15, 30, 0, 0, time.FixedZone("UTC+5", 5*60*60))

func TestHardenedLauncherSpec(t *testing.T) {
	t.Parallel()

	const (
		npmBaseline = "2026-09-20T00:00:00Z" // midnight - 72h (npm floor)
		uvBaseline  = "2026-09-16T00:00:00Z" // midnight - 7d (uv floor)
		strict      = "2026-09-09T00:00:00Z" // midnight - 14d
	)
	env := map[string]string{"K": "${K}"}
	npx := catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0", "."}, Env: env, Transport: "stdio"}
	uvx := catalog.MCPServerDef{Command: "uvx", Args: []string{"--from", "srv==1.0.0", "srv"}, Version: "1.0.0", Bin: "srv"}
	tests := []struct {
		name   string
		def    catalog.MCPServerDef
		minAge time.Duration
		now    time.Time
		want   LaunchSpec
	}{
		{
			name: "npx baseline",
			def:  npx,
			now:  hardenNow,
			want: LaunchSpec{Command: "npx", Args: []string{"pkg@1.0.0", "."}, Env: map[string]string{
				"K": "${K}", npmIgnoreScriptsEnv: "true", npmBeforeEnv: npmBaseline,
			}},
		},
		{
			name:   "npx gate below the floor keeps the floor",
			def:    npx,
			minAge: 24 * time.Hour,
			now:    hardenNow,
			want: LaunchSpec{Command: "npx", Args: []string{"pkg@1.0.0", "."}, Env: map[string]string{
				"K": "${K}", npmIgnoreScriptsEnv: "true", npmBeforeEnv: npmBaseline,
			}},
		},
		{
			name:   "npx strict",
			def:    npx,
			minAge: 14 * 24 * time.Hour,
			now:    hardenNow,
			want: LaunchSpec{Command: "npx", Args: []string{"pkg@1.0.0", "."}, Env: map[string]string{
				"K": "${K}", npmIgnoreScriptsEnv: "true", npmBeforeEnv: strict,
			}},
		},
		{
			name: "npx before UTC midnight counts from the previous UTC day",
			def:  catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}},
			now:  time.Date(2026, 9, 23, 2, 0, 0, 0, time.FixedZone("UTC+5", 5*60*60)),
			want: LaunchSpec{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{
				npmIgnoreScriptsEnv: "true", npmBeforeEnv: "2026-09-19T00:00:00Z",
			}},
		},
		{
			name: "npx allowing install scripts",
			def:  catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, AllowInstallScripts: true},
			now:  hardenNow,
			want: LaunchSpec{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: npmBaseline}},
		},
		{
			name: "windows npx shim",
			def:  catalog.MCPServerDef{Command: `C:\nodejs\npx.cmd`, Args: []string{"pkg@1.0.0"}},
			now:  hardenNow,
			want: LaunchSpec{Command: `C:\nodejs\npx.cmd`, Args: []string{"pkg@1.0.0"}, Env: map[string]string{
				npmIgnoreScriptsEnv: "true", npmBeforeEnv: npmBaseline,
			}},
		},
		{
			name: "absolute npx path",
			def:  catalog.MCPServerDef{Command: "/usr/bin/npx", Args: []string{"pkg@1.0.0"}},
			now:  hardenNow,
			want: LaunchSpec{Command: "/usr/bin/npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{
				npmIgnoreScriptsEnv: "true", npmBeforeEnv: npmBaseline,
			}},
		},
		{
			name: "uvx baseline",
			def:  uvx,
			now:  hardenNow,
			want: LaunchSpec{Command: "uvx", Args: []string{"--exclude-newer=" + uvBaseline, "--from", "srv==1.0.0", "srv"}},
		},
		{
			name:   "uvx strict",
			def:    uvx,
			minAge: 14 * 24 * time.Hour,
			now:    hardenNow,
			want:   LaunchSpec{Command: "uvx", Args: []string{"--exclude-newer=" + strict, "--from", "srv==1.0.0", "srv"}},
		},
		{
			name: "uvx without --from",
			def:  catalog.MCPServerDef{Command: "/usr/bin/uvx", Args: []string{"srv==1.0.0", "--db", "x"}},
			now:  hardenNow,
			want: LaunchSpec{Command: "/usr/bin/uvx", Args: []string{"--exclude-newer=" + uvBaseline, "srv==1.0.0", "--db", "x"}},
		},
		{
			name: "installed bin unchanged",
			def:  catalog.MCPServerDef{Command: "srv", Args: []string{"--db", "x"}, Env: env},
			now:  hardenNow,
			want: LaunchSpec{Command: "srv", Args: []string{"--db", "x"}, Env: env},
		},
		{
			name: "qsdev unchanged",
			def:  catalog.MCPServerDef{Command: "qsdev", Args: []string{"mcp", "serve"}},
			now:  hardenNow,
			want: LaunchSpec{Command: "qsdev", Args: []string{"mcp", "serve"}},
		},
		{
			name: "http unchanged",
			def:  catalog.MCPServerDef{URL: "https://mcp.example.invalid/", Transport: "http", Env: env},
			now:  hardenNow,
			want: LaunchSpec{URL: "https://mcp.example.invalid/", Env: env},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HardenedLauncherSpec(tc.def, tc.minAge, tc.now); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("HardenedLauncherSpec =\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}

// TestHardenedLauncherSpec_StrictCutoffsAtLeastGateOld: whatever the time of
// day, a strict (14-day) launcher admits no release younger than 14 days.
func TestHardenedLauncherSpec_StrictCutoffsAtLeastGateOld(t *testing.T) {
	t.Parallel()
	const gate = 14 * 24 * time.Hour
	for h := range 24 {
		now := time.Date(2026, 9, 23, h, 59, 59, 0, time.UTC)
		npm := HardenedLauncherSpec(catalog.MCPServerDef{Command: "npx", Args: []string{"p@1.0.0"}}, gate, now)
		uv := HardenedLauncherSpec(catalog.MCPServerDef{Command: "uvx", Args: []string{"p==1.0.0"}}, gate, now)
		uvCutoff, _ := strings.CutPrefix(uv.Args[0], uvExcludeNewerFlag)
		for _, v := range []string{npm.Env[npmBeforeEnv], uvCutoff} {
			c, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t.Fatalf("cutoff %q: %v", v, err)
			}
			if c.After(now.Add(-gate)) {
				t.Errorf("at %s cutoff %s is younger than the 14-day gate", now, v)
			}
		}
	}
}

// TestHardenedLauncherSpec_DoesNotAliasDef: the hardened spec owns its env
// and args, so neither hardening nor a caller's edit reaches the catalog.
func TestHardenedLauncherSpec_DoesNotAliasDef(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"npx", "uvx"} {
		env := map[string]string{"K": "v"}
		args := make([]string, 1, 8)
		args[0] = "pkg@1.0.0"
		def := catalog.MCPServerDef{Command: command, Args: args, Env: env}
		got := HardenedLauncherSpec(def, 0, hardenNow)
		got.Args[len(got.Args)-1] = "mutated"
		got.Env["K"] = "mutated"
		if !maps.Equal(env, map[string]string{"K": "v"}) {
			t.Errorf("%s: def.Env changed to %v", command, env)
		}
		if args[0] != "pkg@1.0.0" || slices.Contains(args[:cap(args)], "mutated") {
			t.Errorf("%s: def.Args backing array changed: %v", command, args[:cap(args)])
		}
	}
}

// TestHardenedLauncherSpec_SameDayIsStable: regenerating on the same UTC day
// writes byte-identical cutoffs.
func TestHardenedLauncherSpec_SameDayIsStable(t *testing.T) {
	t.Parallel()
	for _, def := range []catalog.MCPServerDef{
		{Command: "npx", Args: []string{"p@1.0.0"}},
		{Command: "uvx", Args: []string{"p==1.0.0"}},
	} {
		morning := HardenedLauncherSpec(def, 0, time.Date(2026, 9, 23, 0, 0, 1, 0, time.UTC))
		night := HardenedLauncherSpec(def, 0, time.Date(2026, 9, 23, 23, 59, 59, 0, time.UTC))
		if !reflect.DeepEqual(morning, night) {
			t.Errorf("same-day %s specs differ:\n%#v\n%#v", def.Command, morning, night)
		}
	}
}

// TestMatchesTrusted_Hardened: a hardened catalog launcher matches its
// LaunchVariants with any valid cutoff at least the manager's floor old,
// the unhardened launcher still matches, and anything that would loosen the
// hardening does not.
func TestMatchesTrusted_Hardened(t *testing.T) {
	t.Parallel()

	npxDef := catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{"K": "${K}"}}
	uvxDef := catalog.MCPServerDef{Command: "uvx", Args: []string{"--from", "srv==1.0.0", "srv"}}
	scriptsDef := catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, AllowInstallScripts: true}
	npxEnv := func(kv ...string) map[string]string {
		m := map[string]string{"K": "${K}"}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	}
	npxCfg := func(env map[string]string) mcphealth.ServerConfig {
		return mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: env}
	}
	uvxCfg := func(first ...string) mcphealth.ServerConfig {
		return mcphealth.ServerConfig{Command: "uvx", Args: slices.Concat(first, []string{"--from", "srv==1.0.0", "srv"})}
	}
	generatedNpx := HardenedLauncherSpec(npxDef, 14*24*time.Hour, hardenNow)
	generatedUvx := HardenedLauncherSpec(uvxDef, 0, hardenNow)

	tests := []struct {
		name string
		def  catalog.MCPServerDef
		cfg  mcphealth.ServerConfig
		want bool
	}{
		{"npx generated", npxDef, mcphealth.ServerConfig{Command: generatedNpx.Command, Args: generatedNpx.Args, Env: generatedNpx.Env}, true},
		{"npx any valid cutoff", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "2025-01-02T03:04:05Z")), true},
		{"npx cutoff with offset", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "2025-01-02T03:04:05+02:00")), true},
		{"npx unhardened (back-compat)", npxDef, npxCfg(npxEnv()), true},
		{"npx malformed cutoff", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "yesterday")), false},
		{"npx date-only cutoff", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "2025-01-02")), false},
		{"npx future cutoff disables the gate", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "2999-01-01T00:00:00Z")), false},
		{"npx cutoff later than now minus the floor", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339))), false},
		{"npx placeholder literal", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, releaseCutoffPlaceholder)), false},
		{"npx extra env key", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "true", npmBeforeEnv, "2025-01-02T03:04:05Z", "NODE_OPTIONS", "--require=x")), false},
		{"npx missing ignore_scripts", npxDef, npxCfg(npxEnv(npmBeforeEnv, "2025-01-02T03:04:05Z")), false},
		{"npx ignore_scripts false", npxDef, npxCfg(npxEnv(npmIgnoreScriptsEnv, "false", npmBeforeEnv, "2025-01-02T03:04:05Z")), false},
		{"npx scripts allowed by catalog", scriptsDef, mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: "2025-01-02T03:04:05Z"}}, true},
		{"uvx generated", uvxDef, mcphealth.ServerConfig{Command: generatedUvx.Command, Args: generatedUvx.Args}, true},
		{"uvx any valid cutoff", uvxDef, uvxCfg("--exclude-newer=2025-01-02T03:04:05Z"), true},
		{"uvx unhardened (back-compat)", uvxDef, uvxCfg(), true},
		{"uvx malformed cutoff", uvxDef, uvxCfg("--exclude-newer=2025"), false},
		{"uvx placeholder literal", uvxDef, uvxCfg(uvExcludeNewerFlag + releaseCutoffPlaceholder), false},
		{"uvx future cutoff disables the gate", uvxDef, uvxCfg("--exclude-newer=2999-01-01T00:00:00Z"), false},
		{"uvx two-token form", uvxDef, uvxCfg("--exclude-newer", "2025-01-02T03:04:05Z"), false},
		{"uvx cutoff in the wrong position", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{"--from", "srv==1.0.0", "--exclude-newer=2025-01-02T03:04:05Z", "srv"}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesTrusted(tc.cfg, LaunchVariants(tc.def)); got != tc.want {
				t.Errorf("MatchesTrusted(%#v) = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}

// TestMatchesTrusted_LiteralCutoffSpecStillMatches: a trusted spec that
// itself carries a literal cutoff (an embedder-configured server) matches its
// entry exactly as before cutoff canonicalisation existed, whatever the
// cutoff's form or age; it does not match a different cutoff.
func TestMatchesTrusted_LiteralCutoffSpecStillMatches(t *testing.T) {
	t.Parallel()
	for _, cutoff := range []string{"2026-01-01", "2999-01-01T00:00:00Z", "2025-01-02T03:04:05Z"} {
		specs := []LaunchSpec{
			{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: cutoff}},
			{Command: "uvx", Args: []string{uvExcludeNewerFlag + cutoff, "srv==1.0.0"}},
		}
		for _, s := range specs {
			cfg := mcphealth.ServerConfig{Command: s.Command, Args: slices.Clone(s.Args), Env: maps.Clone(s.Env)}
			if !MatchesTrusted(cfg, specs) {
				t.Errorf("literal-cutoff spec %#v does not match itself", s)
			}
		}
	}
	specs := []LaunchSpec{{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: "2025-01-02T03:04:05Z"}}}
	other := mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: "2024-01-02T03:04:05Z"}}
	if MatchesTrusted(other, specs) {
		t.Error("a different cutoff matches a literal-cutoff spec")
	}
}

// TestMatchesTrusted_DoesNotMutateConfig: canonicalising the cutoff works on
// a copy, so the probe still expands the entry as written.
func TestMatchesTrusted_DoesNotMutateConfig(t *testing.T) {
	t.Parallel()
	for _, def := range []catalog.MCPServerDef{
		{Command: "npx", Args: []string{"pkg@1.0.0"}},
		{Command: "uvx", Args: []string{"srv==1.0.0"}},
	} {
		spec := HardenedLauncherSpec(def, 0, hardenNow)
		cfg := mcphealth.ServerConfig{Command: spec.Command, Args: slices.Clone(spec.Args), Env: maps.Clone(spec.Env)}
		if !MatchesTrusted(cfg, LaunchVariants(def)) || !MatchesTrustedLaunch(cfg, LaunchVariants(def)) {
			t.Fatalf("generated hardened %s entry does not match", def.Command)
		}
		if !maps.Equal(cfg.Env, spec.Env) || !slices.Equal(cfg.Args, spec.Args) {
			t.Errorf("matching rewrote the config: %#v", cfg)
		}
	}
}

// TestMatchesTrustedLaunch: the launch-only match compares what runs (the
// command line, or the URL) with the same cutoff canonicalisation as
// MatchesTrusted, and ignores env and headers.
func TestMatchesTrustedLaunch(t *testing.T) {
	t.Parallel()

	npxDef := catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{"K": "${K}"}}
	uvxDef := catalog.MCPServerDef{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}, Version: "1.0.0", Bin: "srv", BinArgs: []string{"--db", "x"}}
	httpDef := catalog.MCPServerDef{URL: "https://mcp.example.invalid/mcp", Transport: "http", Env: map[string]string{"K": "${K}"}}
	tests := []struct {
		name string
		def  catalog.MCPServerDef
		cfg  mcphealth.ServerConfig
		want bool
	}{
		{"npx unhardened", npxDef, mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}}, true},
		{"npx hardened, env ignored", npxDef, mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{npmBeforeEnv: "2025-01-02T03:04:05Z"}}, true},
		{"npx user-added env key", npxDef, mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: map[string]string{"EXTRA": "1"}}, true},
		{"npx extra arg", npxDef, mcphealth.ServerConfig{Command: "npx", Args: []string{"pkg@1.0.0", "-x"}}, false},
		{"uvx hardened", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{"--exclude-newer=2025-01-02T03:04:05Z", "srv==1.0.0", "--db", "x"}}, true},
		{"uvx unhardened", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}}, true},
		{"uvx malformed cutoff", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{"--exclude-newer=soon", "srv==1.0.0", "--db", "x"}}, false},
		{"uvx future cutoff", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{"--exclude-newer=2999-01-01T00:00:00Z", "srv==1.0.0", "--db", "x"}}, false},
		{"uvx placeholder literal", uvxDef, mcphealth.ServerConfig{Command: "uvx", Args: []string{uvExcludeNewerFlag + releaseCutoffPlaceholder, "srv==1.0.0", "--db", "x"}}, false},
		{"installed binary", uvxDef, mcphealth.ServerConfig{Command: "srv", Args: []string{"--db", "x"}}, true},
		{"foreign command", uvxDef, mcphealth.ServerConfig{Command: "evil", Args: []string{"srv==1.0.0", "--db", "x"}}, false},
		{"http, headers and env ignored", httpDef, mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X": "y"}}, true},
		{"http different url", httpDef, mcphealth.ServerConfig{URL: "https://evil.invalid/mcp"}, false},
		{"empty config", npxDef, mcphealth.ServerConfig{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesTrustedLaunch(tc.cfg, LaunchVariants(tc.def)); got != tc.want {
				t.Errorf("MatchesTrustedLaunch(%#v) = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}
