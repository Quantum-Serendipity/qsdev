package golang_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*golang.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &golang.Module{}, "go", "Go", 1)
}

func TestDetect_GoModPresent(t *testing.T) {
	dir := t.TempDir()
	goMod := "module example.com/foo\n\ngo 1.22.5\n\nrequire (\n\tgolang.org/x/text v0.14.0\n)\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &golang.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when go.mod is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if result.SuggestedConfig.Version != "1.22.5" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "1.22.5")
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	foundGoMod := false
	foundVersion := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "go.mod") {
			foundGoMod = true
		}
		if strings.Contains(e, "1.22.5") {
			foundVersion = true
		}
	}
	if !foundGoMod {
		t.Error("evidence should mention go.mod")
	}
	if !foundVersion {
		t.Error("evidence should mention the detected version")
	}
}

func TestDetect_GoModMinorOnly(t *testing.T) {
	dir := t.TempDir()
	goMod := "module example.com/bar\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &golang.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "1.22" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "1.22")
	}
}

func TestDetect_GoModNoDirective(t *testing.T) {
	dir := t.TempDir()
	goMod := "module example.com/baz\n\nrequire (\n\tgolang.org/x/text v0.14.0\n)\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &golang.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true even without go directive")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if result.SuggestedConfig.Version != "" {
		t.Errorf("Version = %q, want empty string", result.SuggestedConfig.Version)
	}
}

func TestDetect_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	m := &golang.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no go.mod")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &golang.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	requiredStrings := []string{
		"languages.go.enable = true;",
		`env.GOSUMDB = "sum.golang.org";`,
	}

	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
}

// TestDevenvNixFragment_NoGOFLAGS guards against GOFLAGS=-mod=readonly: it
// is already the build default, does not stop `go get`/`go mod tidy` from
// adding dependencies, and overrides the -mod=vendor default so builds of a
// vendored module silently bypass the reviewed vendor/ tree.
func TestDevenvNixFragment_NoGOFLAGS(t *testing.T) {
	t.Parallel()
	for _, cfg := range []ecosystem.ModuleConfig{{}, {Version: "1.22"}, {RegistryProxy: "https://goproxy.corp.example.com"}} {
		fragment, err := (&golang.Module{}).DevenvNixFragment(cfg)
		if err != nil {
			t.Fatalf("DevenvNixFragment() returned error: %v", err)
		}
		if strings.Contains(fragment, "GOFLAGS") || strings.Contains(fragment, "-mod=") {
			t.Errorf("fragment sets GOFLAGS/-mod, which overrides vendor mode:\n%s", fragment)
		}
	}
}

// TestDevenvNixFragment_EnvVarsAreGoEnvironment guards against emitting
// variables the go command does not read (GONOSUMCHECK) or no-op settings
// (an empty GONOSUMDB is identical to unset). Every emitted env var must be
// one listed by `go help environment`.
func TestDevenvNixFragment_EnvVarsAreGoEnvironment(t *testing.T) {
	t.Parallel()
	known := map[string]bool{
		"GOFLAGS": true, "GOPROXY": true, "GOSUMDB": true, "GONOSUMDB": true,
		"GOPRIVATE": true, "GONOPROXY": true, "GOTOOLCHAIN": true, "GOINSECURE": true,
	}
	envLineRe := regexp.MustCompile(`(?m)^\s*env\.([A-Z0-9_]+) = (.*);$`)
	for _, proxy := range []string{"", "https://goproxy.corp.example.com"} {
		fragment, err := (&golang.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{RegistryProxy: proxy})
		if err != nil {
			t.Fatalf("DevenvNixFragment() returned error: %v", err)
		}
		matches := envLineRe.FindAllStringSubmatch(fragment, -1)
		if len(matches) == 0 {
			t.Fatalf("no env vars emitted:\n%s", fragment)
		}
		for _, m := range matches {
			if !known[m[1]] {
				t.Errorf("env.%s is not a go environment variable\ngot:\n%s", m[1], fragment)
			}
			if m[2] == `""` {
				t.Errorf("env.%s is set to an empty string, which the go command treats as unset", m[1])
			}
		}
	}
}

// TestDevenvNixFragment_VersionMapping verifies the go.mod version becomes
// a minimum: nixpkgs' Go is kept when it is new enough, and otherwise the
// exact release comes from go-overlay (devenv sets GOTOOLCHAIN=local, so an
// older Go refuses to build the module). No go_1_X attribute is ever named,
// since nixpkgs removes them as Go releases reach end of life.
func TestDevenvNixFragment_VersionMapping(t *testing.T) {
	t.Parallel()
	m := &golang.Module{}

	tests := []struct {
		name        string
		config      ecosystem.ModuleConfig
		wantRelease string // "" means no version line: nixpkgs' Go
		wantNote    bool
	}{
		{"empty version uses nixpkgs go", ecosystem.ModuleConfig{}, "", false},
		{"bare minor is its first release", ecosystem.ModuleConfig{Version: "1.24"}, "1.24.0", false},
		{"eol minor still resolves", ecosystem.ModuleConfig{Version: "1.23"}, "1.23.0", false},
		{"patch release kept exactly", ecosystem.ModuleConfig{Version: "1.26.8"}, "1.26.8", false},
		{"pre-1.21 minor has no .0", ecosystem.ModuleConfig{Version: "1.20"}, "1.20", false},
		{"pre-1.21 .0 is the bare release", ecosystem.ModuleConfig{Version: "1.20.0"}, "1.20", false},
		{"release candidate", ecosystem.ModuleConfig{Version: "1.27rc1"}, "1.27rc1", false},
		{"toolchain directive wins", ecosystem.ModuleConfig{Version: "1.24", Extras: map[string]string{golang.ExtraToolchain: "1.26.8"}}, "1.26.8", false},
		// --go-version sets Version while detection still supplies the
		// toolchain extra: an explicit newer version must not be dropped.
		{"newer explicit version beats toolchain", ecosystem.ModuleConfig{Version: "1.27.1", Extras: map[string]string{golang.ExtraToolchain: "1.26.8"}}, "1.27.1", false},
		{"invalid toolchain falls back to version", ecosystem.ModuleConfig{Version: "1.25.3", Extras: map[string]string{golang.ExtraToolchain: "1.26\nbuiltins.abort"}}, "1.25.3", true},
		{"single component ignored", ecosystem.ModuleConfig{Version: "1"}, "", true},
		{"non-numeric input ignored", ecosystem.ModuleConfig{Version: "1.x; builtins.abort"}, "", true},
		{"go 2 ignored", ecosystem.ModuleConfig{Version: "2.0"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fragment, err := m.DevenvNixFragment(tt.config)
			if err != nil {
				t.Fatalf("DevenvNixFragment() returned error: %v", err)
			}
			if strings.Contains(fragment, "package =") || strings.Contains(fragment, "go_1_") {
				t.Errorf("fragment must not pin a nixpkgs go_1_X attribute:\n%s", fragment)
			}
			if tt.wantRelease == "" {
				if strings.Contains(fragment, "version =") {
					t.Errorf("unexpected version line:\n%s", fragment)
				}
			} else {
				want := `version = lib.mkIf (!(lib.versionAtLeast pkgs.go.version "` + tt.wantRelease + `")) "` + tt.wantRelease + `";`
				if !strings.Contains(fragment, want) {
					t.Errorf("fragment missing %s\ngot:\n%s", want, fragment)
				}
			}
			if hasNote := strings.HasPrefix(fragment, "  # "); hasNote != tt.wantNote {
				t.Errorf("note present = %v, want %v\ngot:\n%s", hasNote, tt.wantNote, fragment)
			}
			// go.mod is repo-controlled: its text may only appear in comments.
			for line := range strings.SplitSeq(fragment, "\n") {
				if strings.Contains(line, "abort") && !strings.HasPrefix(line, "  # ") {
					t.Errorf("untrusted version text escaped the comment: %q", line)
				}
			}
			// The version option needs the go-overlay input, and only then.
			inputs := m.DevenvYamlInputs(tt.config)
			if hasInput := len(inputs) == 1 && inputs[0].URL == "github:purpleclay/go-overlay" && inputs[0].Follows == "nixpkgs"; hasInput != (tt.wantRelease != "") {
				t.Errorf("DevenvYamlInputs() = %+v, want go-overlay input = %v", inputs, tt.wantRelease != "")
			}
		})
	}
}

// TestDetect_GoModToolchain verifies the toolchain directive is detected,
// since it names the release the project actually builds with.
func TestDetect_GoModToolchain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		goMod         string
		wantVersion   string
		wantToolchain string
	}{
		{"toolchain after go", "module x\n\ngo 1.24\n\ntoolchain go1.26.8\n", "1.24", "1.26.8"},
		{"release candidate toolchain", "module x\n\ngo 1.26.0\ntoolchain go1.27rc1\n", "1.26.0", "1.27rc1"},
		{"no toolchain", "module x\n\ngo 1.25.3\n", "1.25.3", ""},
		{"custom toolchain ignored", "module x\n\ngo 1.25.3\ntoolchain default\n", "1.25.3", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(tt.goMod), 0o644); err != nil {
				t.Fatal(err)
			}
			r := (&golang.Module{}).Detect(dir)
			if r.SuggestedConfig.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", r.SuggestedConfig.Version, tt.wantVersion)
			}
			if got := r.SuggestedConfig.Extra(golang.ExtraToolchain, ""); got != tt.wantToolchain {
				t.Errorf("toolchain = %q, want %q", got, tt.wantToolchain)
			}
		})
	}
}

func TestDevenvNixFragment_RegistryProxy(t *testing.T) {
	m := &golang.Module{}
	proxy := "https://goproxy.corp.example.com"
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		RegistryProxy: proxy,
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	// No ",direct": the go command falls back to the next entry on any
	// 404/410, so a proxy refusing a module would be silently bypassed.
	expected := `env.GOPROXY = "` + proxy + `";`
	if !strings.Contains(fragment, expected) {
		t.Errorf("DevenvNixFragment() missing GOPROXY line\nwant: %s\ngot:\n%s", expected, fragment)
	}
	if strings.Contains(fragment, ",direct") {
		t.Errorf("GOPROXY must not fall back to direct:\n%s", fragment)
	}
	// Existing security settings must be preserved.
	for _, s := range []string{"GOSUMDB"} {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q when proxy is set\ngot:\n%s", s, fragment)
		}
	}
}

func TestDevenvNixFragment_NoRegistryProxy(t *testing.T) {
	m := &golang.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if strings.Contains(fragment, "GOPROXY") {
		t.Errorf("DevenvNixFragment() should not contain GOPROXY when proxy is empty\ngot:\n%s", fragment)
	}
}

func TestDevenvNixFragment_RegistryProxyPreservesExisting(t *testing.T) {
	m := &golang.Module{}
	proxy := "https://goproxy.corp.example.com"
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		RegistryProxy: proxy,
		Version:       "1.22",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	// All existing env vars must still be present.
	for _, s := range []string{
		`env.GOSUMDB = "sum.golang.org"`,
		"languages.go",
		"enable = true",
	} {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q when proxy is set\ngot:\n%s", s, fragment)
		}
	}
}

func TestSecurityConfigs(t *testing.T) {
	m := &golang.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})
	if configs != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", configs)
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &golang.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 4 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 4", len(hooks))
	}

	expectedIDs := []string{"gofmt", "govet", "staticcheck", "govulncheck"}
	expectedBuiltIn := []bool{true, true, false, false}

	for i, hook := range hooks {
		if hook.ID != expectedIDs[i] {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, expectedIDs[i])
		}
		if hook.BuiltIn != expectedBuiltIn[i] {
			t.Errorf("hooks[%d].BuiltIn = %v, want %v", i, hook.BuiltIn, expectedBuiltIn[i])
		}
		if hook.Language != "system" {
			t.Errorf("hooks[%d].Language = %q, want %q", i, hook.Language, "system")
		}
		if len(hook.Types) != 1 || hook.Types[0] != "go" {
			t.Errorf("hooks[%d].Types = %v, want [\"go\"]", i, hook.Types)
		}
	}
}

func TestCICommands(t *testing.T) {
	m := &golang.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 3 {
		t.Fatalf("CICommands() returned %d commands, want 3", len(cmds))
	}

	expectedPhases := []ecosystem.CIPhase{
		ecosystem.CIPhaseInstall,
		ecosystem.CIPhaseTest,
		ecosystem.CIPhaseScan,
	}
	expectedCommands := []string{
		"go mod download",
		"go mod verify",
		"govulncheck ./...",
	}

	for i, cmd := range cmds {
		if cmd.Phase != expectedPhases[i] {
			t.Errorf("cmds[%d].Phase = %v, want %v", i, cmd.Phase, expectedPhases[i])
		}
		if cmd.Command != expectedCommands[i] {
			t.Errorf("cmds[%d].Command = %q, want %q", i, cmd.Command, expectedCommands[i])
		}
		if cmd.Name == "" {
			t.Errorf("cmds[%d].Name should not be empty", i)
		}
		if cmd.Description == "" {
			t.Errorf("cmds[%d].Description should not be empty", i)
		}
	}
}

func TestPackageManagers(t *testing.T) {
	m := &golang.Module{}
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}

	pm := pms[0]
	if pm.Name != "go modules" {
		t.Errorf("Name = %q, want %q", pm.Name, "go modules")
	}
	if pm.LockFile != "go.sum" {
		t.Errorf("LockFile = %q, want %q", pm.LockFile, "go.sum")
	}
	if pm.FrozenInstallCommand != "go mod download" {
		t.Errorf("FrozenInstallCommand = %q, want %q", pm.FrozenInstallCommand, "go mod download")
	}
	if pm.AuditCommand != "govulncheck ./..." {
		t.Errorf("AuditCommand = %q, want %q", pm.AuditCommand, "govulncheck ./...")
	}
	if pm.AgeGatingSupport {
		t.Error("AgeGatingSupport should be false")
	}
}

func TestWizardFields(t *testing.T) {
	m := &golang.Module{}
	fields := m.WizardFields()

	if len(fields) != 1 {
		t.Fatalf("WizardFields() returned %d fields, want 1", len(fields))
	}

	f := fields[0]
	if f.Key != "go_version" {
		t.Errorf("Key = %q, want %q", f.Key, "go_version")
	}
	if f.Type != ecosystem.FieldTypeInput {
		t.Errorf("Type = %v, want FieldTypeInput", f.Type)
	}
}

func TestDevenvPackages(t *testing.T) {
	t.Parallel()
	var m ecosystem.EcosystemModule = &golang.Module{}
	pp, ok := m.(ecosystem.PackageProvider)
	if !ok {
		t.Fatal("Go module does not implement PackageProvider")
	}
	pkgs := pp.DevenvPackages(ecosystem.ModuleConfig{})

	want := map[string]bool{"gopls": true, "golangci-lint": true, "delve": true, "goreleaser": true}
	got := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		got[p] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("DevenvPackages() missing %q; got %v", name, pkgs)
		}
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("go")
	if !ok {
		t.Fatal("expected module 'go' to be registered in DefaultRegistry")
	}
	if mod.Name() != "go" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "go")
	}
}
