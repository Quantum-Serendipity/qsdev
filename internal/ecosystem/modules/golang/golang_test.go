package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/golang"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*golang.Module)(nil)

func TestName(t *testing.T) {
	m := &golang.Module{}
	if got := m.Name(); got != "go" {
		t.Errorf("Name() = %q, want %q", got, "go")
	}
}

func TestDisplayName(t *testing.T) {
	m := &golang.Module{}
	if got := m.DisplayName(); got != "Go" {
		t.Errorf("DisplayName() = %q, want %q", got, "Go")
	}
}

func TestTier(t *testing.T) {
	m := &golang.Module{}
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want %d", got, 1)
	}
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
		"languages.go",
		"enable = true",
		"package = pkgs.go",
		"GOFLAGS",
		`"-mod=readonly"`,
		"GONOSUMCHECK",
		"GONOSUMDB",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
}

func TestDevenvYamlInputs(t *testing.T) {
	m := &golang.Module{}
	inputs := m.DevenvYamlInputs(ecosystem.ModuleConfig{})
	if inputs != nil {
		t.Errorf("DevenvYamlInputs() = %v, want nil", inputs)
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

func TestDenyRules(t *testing.T) {
	m := &golang.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}

	expected := []string{
		"Bash(go get *)",
		"Bash(go install *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
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
