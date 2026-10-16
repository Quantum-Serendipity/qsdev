package claudecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeInstalledMCPState records servers as installed in the claude state
// file under root, as `qsdev mcp install` does.
func writeInstalledMCPState(t *testing.T, root string, servers map[string]types.McpServerState) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(statePath()))
	s := types.GeneratedState{Files: map[string]types.FileState{}, McpServers: servers}
	if err := state.SaveStateToFile(path, s); err != nil {
		t.Fatalf("writing state: %v", err)
	}
}

// installedAtPin returns an install record at the pinned release for every
// catalog server whose install method provides a binary.
func installedAtPin(cat *catalog.Catalog) map[string]types.McpServerState {
	out := make(map[string]types.McpServerState)
	for name, def := range cat.MCPServers() {
		if def.Bin != "" {
			out[name] = types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}
		}
	}
	return out
}

func generatedEntries(t *testing.T, answers types.WizardAnswers, cfg Config) map[string]MCPServerEntry {
	t.Helper()
	f, err := GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("GenerateMcpJson: %v", err)
	}
	var mcp McpJSON
	if err := json.Unmarshal(f.Content, &mcp); err != nil {
		t.Fatalf("parsing .mcp.json: %v", err)
	}
	return mcp.MCPServers
}

// TestGenerateMcpJson_RefusesUnpinnedLaunchers covers F082: a server that
// would fetch a package at launch without an exact version is refused,
// whichever source configured it.
func TestGenerateMcpJson_RefusesUnpinnedLaunchers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{name: "npx auto-confirm latest", command: "npx", args: []string{"-y", "@upstash/context7-mcp"}, wantErr: true},
		{name: "npx range", command: "npx", args: []string{"some-mcp@^1.2.0"}, wantErr: true},
		{name: "npx dist-tag", command: "npx", args: []string{"some-mcp@latest"}, wantErr: true},
		{name: "uvx unpinned from", command: "uvx", args: []string{"--from", "semble[mcp]", "semble"}, wantErr: true},
		{name: "uvx bare package", command: "uvx", args: []string{"mcp-nixos"}, wantErr: true},
		{name: "uvx wildcard", command: "uvx", args: []string{"--from", "pkg==1.*", "pkg"}, wantErr: true},
		{name: "absolute npx path", command: "/usr/bin/npx", args: []string{"some-mcp"}, wantErr: true},
		{name: "env wrapped uvx", command: "env", args: []string{"FOO=1", "uvx", "pkg"}, wantErr: true},
		{name: "pnpm dlx", command: "pnpm", args: []string{"dlx", "some-mcp"}, wantErr: true},
		{name: "docker without digest", command: "docker", args: []string{"run", "-i", "mcp/server:latest"}, wantErr: true},
		{name: "npx exact", command: "npx", args: []string{"some-mcp@1.2.3"}},
		{name: "uvx exact", command: "uvx", args: []string{"--from", "semble[mcp]==0.6.0", "semble"}},
		{name: "local binary", command: "my-mcp-server", args: []string{"--stdio"}},
		{name: "pnpm local script", command: "pnpm", args: []string{"exec", "tsx", "server.ts"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := NewConfig(WithMCPServer(MCPServerConfig{Name: "custom", Command: tt.command, Args: tt.args}))
			_, err := GenerateMcpJson(types.WizardAnswers{}, cfg)
			if got := errors.Is(err, ErrUnpinnedMCPServer); got != tt.wantErr {
				t.Fatalf("err = %v, want ErrUnpinnedMCPServer: %v", err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestCatalogMCPServers_Pinned covers F082 for the catalog: every server
// generates (so no launcher is unpinned), and a server with an install
// method pins the same exact release in its launcher and for `qsdev mcp
// install`.
func TestCatalogMCPServers_Pinned(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	entries := generatedEntries(t, types.WizardAnswers{MCPServers: cat.MCPServerNames()}, Config{})
	for name, def := range cat.MCPServers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, ok := entries[name]; !ok {
				t.Fatalf("%s missing from generated .mcp.json", name)
			}
			var spec string
			switch def.InstallMethod {
			case "npm-global":
				spec = def.PackageName + "@" + def.Version
			case "uv-tool":
				spec = def.PackageName + "==" + def.Version
			default:
				return
			}
			if def.Version == "" || def.Bin == "" {
				t.Fatalf("install method %s without version (%q) or bin (%q)", def.InstallMethod, def.Version, def.Bin)
			}
			if def.Command != def.Bin && !slices.Contains(def.Args, spec) {
				t.Errorf("launcher %s %v does not run the pinned %s", def.Command, def.Args, spec)
			}
			if def.Command != def.Bin && !slices.Equal(def.Args[len(def.Args)-len(def.BinArgs):], def.BinArgs) {
				t.Errorf("launcher args %v do not end with bin_args %v", def.Args, def.BinArgs)
			}
		})
	}
}

// TestGenerateMcpJson_PrefersInstalledBinary covers F082: a server `qsdev mcp
// install` installed at the pinned release runs the installed binary; any
// other record keeps the pinned launcher.
func TestGenerateMcpJson_PrefersInstalledBinary(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	const server = "postgres" // its binary takes arguments (bin_args)
	def, ok := cat.MCPServer(server)
	if !ok || def.Bin == "" {
		t.Fatalf("catalog server %s must provide a binary", server)
	}
	binary := MCPServerEntry{Command: def.Bin, Args: def.BinArgs, Env: def.Env}

	tests := []struct {
		name     string
		record   *types.McpServerState
		noRoot   bool
		launcher bool // want the hardened launcher rather than binary
	}{
		{name: "installed at pin", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}},
		{name: "installed at another version", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: "0.0.1"}, launcher: true},
		{name: "installed by another method", record: &types.McpServerState{InstallMethod: "npm-global", InstalledVersion: def.Version}, launcher: true},
		{name: "not installed", launcher: true},
		{name: "no project root", record: &types.McpServerState{InstallMethod: def.InstallMethod, InstalledVersion: def.Version}, noRoot: true, launcher: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			root := t.TempDir()
			if tt.record != nil {
				writeInstalledMCPState(t, root, map[string]types.McpServerState{server: *tt.record})
			}
			answers := types.WizardAnswers{MCPServers: []string{server}, ProjectRoot: root}
			if tt.noRoot {
				answers.ProjectRoot = ""
			}
			got := generatedEntries(t, answers, Config{})[server]

			// `qsdev enable` writes the same entry into .mcp.json.
			raw, err := mcpServerContentFunc(server)(answers)
			if err != nil {
				t.Fatalf("enable content: %v", err)
			}
			var enabled MCPServerEntry
			if err := json.Unmarshal(raw, &enabled); err != nil {
				t.Fatal(err)
			}

			wants := []MCPServerEntry{binary}
			if tt.launcher {
				wants = hardenedLaunchers(def, 0, start)
			}
			for path, e := range map[string]MCPServerEntry{"generate": got, "enable": enabled} {
				if !slices.ContainsFunc(wants, func(w MCPServerEntry) bool { return sameLaunchEntry(e, w) }) {
					t.Errorf("%s entry = %s %v, want one of %v", path, e.Command, e.Args, wants)
				}
			}
		})
	}
}

// TestSembleTextFiles_InstalledBinary covers F082: text-file indexing appends
// its flag to whichever entry the server runs, so installing semble does not
// fall back to the fetch-on-run launcher.
func TestSembleTextFiles_InstalledBinary(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := cat.MCPServer(sembleServerName)
	if !ok {
		t.Fatal("semble missing from catalog")
	}
	root := t.TempDir()
	writeInstalledMCPState(t, root, installedAtPin(cat))
	answers := types.WizardAnswers{ProjectRoot: root}
	answers.AgentTools.SembleEnabled = true
	answers.AgentTools.SembleMode = "mcp"
	answers.AgentTools.SembleTextFiles = true

	res, err := generateSembleConfig(answers)
	if err != nil {
		t.Fatal(err)
	}
	if res.Override == nil {
		t.Fatal("no semble override for text-file indexing")
	}
	want := append(append([]string{}, def.BinArgs...), "--include-text-files")
	if res.Override.Command != def.Bin || !slices.Equal(res.Override.Args, want) {
		t.Errorf("override = %s %v, want %s %v", res.Override.Command, res.Override.Args, def.Bin, want)
	}
}

// TestSembleTextFilesServer_FromCatalog guards that the text-file flag comes
// from the catalog's optional_args alone, so generation cannot drift from the
// trusted variants: a definition declaring none is an error, not a guess.
func TestSembleTextFilesServer_FromCatalog(t *testing.T) {
	t.Parallel()
	entry := MCPServerEntry{Command: "semble", Args: []string{"serve"}}
	tests := []struct {
		name     string
		optional [][]string
		wantArgs []string
		wantErr  bool
	}{
		{name: "first optional group appended", optional: [][]string{{"--text", "on"}, {"--other"}}, wantArgs: []string{"serve", "--text", "on"}},
		{name: "no optional args", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := sembleTextFilesServer(entry, catalog.MCPServerDef{OptionalArgs: tt.optional})
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !slices.Equal(got.Args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", got.Args, tt.wantArgs)
			}
			if !tt.wantErr && !slices.Equal(entry.Args, []string{"serve"}) {
				t.Errorf("entry args mutated: %v", entry.Args)
			}
		})
	}
}

// TestTrustedDefinitions_CoversGeneratedVariants proves generation and the
// probe trust set share one source: every entry generation can write for a
// catalog server (each server, installed or not, semble text-file indexing on
// or off) is one of mcpregistry.LaunchVariants, and every generated entry is
// trusted by the CLI's probe gate.
func TestTrustedDefinitions_CoversGeneratedVariants(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	trusted := mcpregistry.TrustedDefinitions(configuredServerSpecs())
	configured := make(map[string]bool, len(addon.Config.MCPServers))
	for _, srv := range addon.Config.MCPServers {
		configured[srv.Name] = true
	}
	for _, installed := range []bool{false, true} {
		for _, textFiles := range []bool{false, true} {
			t.Run(fmt.Sprintf("installed=%v/textfiles=%v", installed, textFiles), func(t *testing.T) {
				t.Parallel()
				answers := types.WizardAnswers{MCPServers: cat.MCPServerNames(), ProjectRoot: t.TempDir()}
				if installed {
					writeInstalledMCPState(t, answers.ProjectRoot, installedAtPin(cat))
				}
				answers.AgentTools.SembleEnabled = true
				answers.AgentTools.SembleMode = "mcp"
				answers.AgentTools.SembleTextFiles = textFiles
				sr, err := generateSembleConfig(answers)
				if err != nil {
					t.Fatal(err)
				}
				cfg := addon.Config
				if sr.Override != nil {
					cfg.MCPServers = append(append([]MCPServerConfig{}, cfg.MCPServers...), *sr.Override)
				}
				for name, e := range generatedEntries(t, answers, cfg) {
					sc := mcphealth.ServerConfig{Name: name, Command: e.Command, Args: e.Args, URL: e.URL, Env: e.Env, Headers: e.Headers}
					if !mcpregistry.MatchesTrusted(sc, trusted[name]) {
						t.Errorf("generated %s (%s %v %s) is not trusted", name, e.Command, e.Args, e.URL)
					}
					def, ok := cat.MCPServer(name)
					if !ok || configured[name] {
						continue
					}
					if !mcpregistry.MatchesTrusted(sc, mcpregistry.LaunchVariants(def)) {
						t.Errorf("generated %s (%s %v %s) is not in LaunchVariants", name, e.Command, e.Args, e.URL)
					}
				}
			})
		}
	}
}

// complianceLevels are the levels the hardening tests generate for: the
// lowest gate and one above every launcher floor.
var complianceLevels = []string{"baseline", "strict"}

// hardenedLaunchers returns the hardened launcher entries generation may have
// written for def at gate minAge since start: one per UTC day from start to
// now, which is a single entry unless the test ran past midnight.
func hardenedLaunchers(def catalog.MCPServerDef, minAge time.Duration, start time.Time) []MCPServerEntry {
	return []MCPServerEntry{
		launchSpecEntry(mcpregistry.HardenedLauncherSpec(def, minAge, start)),
		launchSpecEntry(mcpregistry.HardenedLauncherSpec(def, minAge, time.Now())),
	}
}

// sameLaunchEntry reports whether a and b launch the same command line.
func sameLaunchEntry(a, b MCPServerEntry) bool {
	return a.Command == b.Command && slices.Equal(a.Args, b.Args) && a.URL == b.URL
}

// assertCutoff checks that v is an RFC 3339 cutoff at least the effective
// release age (gate raised to floor) before now, and at most a day older:
// generation counts back from today's UTC midnight.
func assertCutoff(t *testing.T, v string, gate, floor time.Duration) {
	t.Helper()
	cutoff, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t.Fatalf("cutoff %q is not RFC 3339: %v", v, err)
	}
	age := ecosystem.EffectiveReleaseAge(gate, floor)
	now := time.Now()
	if cutoff.After(now.Add(-age)) {
		t.Errorf("cutoff %s is later than now - %v", v, age)
	}
	if cutoff.Before(now.Add(-age - 25*time.Hour)) {
		t.Errorf("cutoff %s is more than a day older than now - %v", v, age)
	}
}

// assertNpxHardened checks that e is def's npx launcher with npm's install
// scripts off (unless def allows them) and an age-gated npm_config_before,
// keeping def's own env.
func assertNpxHardened(t *testing.T, e MCPServerEntry, def catalog.MCPServerDef, gate time.Duration) {
	t.Helper()
	if e.Command != def.Command || !slices.Equal(e.Args, def.Args) {
		t.Errorf("entry = %s %v, want the launcher %s %v", e.Command, e.Args, def.Command, def.Args)
	}
	want := "true"
	if def.AllowInstallScripts {
		want = ""
	}
	if got := e.Env["npm_config_ignore_scripts"]; got != want {
		t.Errorf("npm_config_ignore_scripts = %q, want %q", got, want)
	}
	assertCutoff(t, e.Env["npm_config_before"], gate, installer.NpmMinReleaseAge)
	for k, v := range def.Env {
		if e.Env[k] != v {
			t.Errorf("env %s = %q, want the catalog's %q", k, e.Env[k], v)
		}
	}
}

// assertUvxHardened checks that e is def's uvx launcher led by an age-gated
// --exclude-newer= cutoff, followed by def's args and then extra.
func assertUvxHardened(t *testing.T, e MCPServerEntry, def catalog.MCPServerDef, gate time.Duration, extra ...string) {
	t.Helper()
	if e.Command != def.Command || len(e.Args) == 0 {
		t.Fatalf("entry = %s %v, want the uvx launcher", e.Command, e.Args)
	}
	cutoff, ok := strings.CutPrefix(e.Args[0], "--exclude-newer=")
	if !ok {
		t.Fatalf("args %v do not start with --exclude-newer=", e.Args)
	}
	assertCutoff(t, cutoff, gate, installer.UvMinReleaseAge)
	if want := slices.Concat(def.Args, extra); !slices.Equal(e.Args[1:], want) {
		t.Errorf("args after the cutoff = %v, want %v", e.Args[1:], want)
	}
}

// TestCatalogServerEntry_NpxHardened covers U22-07 for npx launchers: the
// generated and the enabled entry both skip install scripts and only resolve
// releases older than the project's gate.
func TestCatalogServerEntry_NpxHardened(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"context7", "local-docs-devdocs", "filesystem"} {
		for _, level := range complianceLevels {
			t.Run(name+"/"+level, func(t *testing.T) {
				t.Parallel()
				def, ok := cat.MCPServer(name)
				if !ok || def.Command != "npx" {
					t.Fatalf("catalog %s is not an npx launcher", name)
				}
				gate := catalog.EffectiveAgeGate(level)
				answers := types.WizardAnswers{MCPServers: []string{name}, ComplianceLevel: level}
				assertNpxHardened(t, generatedEntries(t, answers, Config{})[name], def, gate)

				raw, err := mcpServerContentFunc(name)(answers)
				if err != nil {
					t.Fatalf("enable content: %v", err)
				}
				var enabled MCPServerEntry
				if err := json.Unmarshal(raw, &enabled); err != nil {
					t.Fatal(err)
				}
				assertNpxHardened(t, enabled, def, gate)
			})
		}
	}
}

// TestCatalogServerEntry_UvxExcludeNewer covers U22-07 for uvx launchers,
// including semble's text-file flag, which follows the hardened args.
func TestCatalogServerEntry_UvxExcludeNewer(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fetch", sembleServerName, "mcp-nixos", "postgres"} {
		for _, level := range complianceLevels {
			t.Run(name+"/"+level, func(t *testing.T) {
				t.Parallel()
				def, ok := cat.MCPServer(name)
				if !ok || def.Command != "uvx" {
					t.Fatalf("catalog %s is not a uvx launcher", name)
				}
				gate := catalog.EffectiveAgeGate(level)
				answers := types.WizardAnswers{MCPServers: []string{name}, ComplianceLevel: level}
				assertUvxHardened(t, generatedEntries(t, answers, Config{})[name], def, gate)

				raw, err := mcpServerContentFunc(name)(answers)
				if err != nil {
					t.Fatalf("enable content: %v", err)
				}
				var enabled MCPServerEntry
				if err := json.Unmarshal(raw, &enabled); err != nil {
					t.Fatal(err)
				}
				assertUvxHardened(t, enabled, def, gate)

				if name != sembleServerName {
					return
				}
				answers.AgentTools.SembleEnabled = true
				answers.AgentTools.SembleMode = "mcp"
				answers.AgentTools.SembleTextFiles = true
				res, err := generateSembleConfig(answers)
				if err != nil {
					t.Fatal(err)
				}
				if res.Override == nil {
					t.Fatal("no semble override for text-file indexing")
				}
				o := res.Override
				assertUvxHardened(t, MCPServerEntry{Command: o.Command, Args: o.Args, Env: o.Env}, def, gate, def.OptionalArgs[0]...)
			})
		}
	}
}

// TestGeneratedMcpEntries_HardenedProperty covers U22-07 across the whole
// catalog: at every level, every generated npx entry is script-free and
// age-gated and every uvx entry leads with --exclude-newer=, while an
// installed server runs exactly its bare binary, as before hardening.
func TestGeneratedMcpEntries_HardenedProperty(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range complianceLevels {
		for _, installed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/installed=%v", level, installed), func(t *testing.T) {
				t.Parallel()
				answers := types.WizardAnswers{MCPServers: cat.MCPServerNames(), ComplianceLevel: level, ProjectRoot: t.TempDir()}
				if installed {
					writeInstalledMCPState(t, answers.ProjectRoot, installedAtPin(cat))
				}
				gate := catalog.EffectiveAgeGate(level)
				for name, e := range generatedEntries(t, answers, Config{}) {
					def, ok := cat.MCPServer(name)
					if !ok {
						t.Fatalf("generated %s is not a catalog server", name)
					}
					if spec, ok := mcpregistry.InstalledSpec(def); ok && installed {
						if e.Command != spec.Command || !slices.Equal(e.Args, spec.Args) || !maps.Equal(e.Env, spec.Env) {
							t.Errorf("installed %s = %s %v %v, want the bare %s %v %v", name, e.Command, e.Args, e.Env, spec.Command, spec.Args, spec.Env)
						}
						continue
					}
					switch def.Command {
					case "npx":
						assertNpxHardened(t, e, def, gate)
					case "uvx":
						assertUvxHardened(t, e, def, gate)
					}
				}
			})
		}
	}
}

// TestGenerateMcpJson_SameDayDeterministic covers the cutoff's midnight
// truncation: regenerating .mcp.json within a UTC day is byte-identical, so
// `init --update` leaves an unchanged project's file alone.
func TestGenerateMcpJson_SameDayDeterministic(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range complianceLevels {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{MCPServers: cat.MCPServerNames(), ComplianceLevel: level}
			for {
				start := time.Now()
				first, err := GenerateMcpJson(answers, Config{})
				if err != nil {
					t.Fatal(err)
				}
				second, err := GenerateMcpJson(answers, Config{})
				if err != nil {
					t.Fatal(err)
				}
				if !sameUTCDay(start, time.Now()) {
					continue // the runs straddled midnight; compare a same-day pair
				}
				if !bytes.Equal(first.Content, second.Content) {
					t.Errorf("same-day regeneration differs:\n%s\n---\n%s", first.Content, second.Content)
				}
				return
			}
		})
	}
}

// sameUTCDay reports whether a and b fall on the same UTC calendar day.
func sameUTCDay(a, b time.Time) bool {
	return a.UTC().Truncate(24 * time.Hour).Equal(b.UTC().Truncate(24 * time.Hour))
}
