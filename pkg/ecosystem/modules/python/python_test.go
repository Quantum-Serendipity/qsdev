package python_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*python.Module)(nil)

func TestName(t *testing.T) {
	m := &python.Module{}
	if got := m.Name(); got != "python" {
		t.Errorf("Name() = %q, want %q", got, "python")
	}
}

func TestDisplayName(t *testing.T) {
	m := &python.Module{}
	if got := m.DisplayName(); got != "Python" {
		t.Errorf("DisplayName() = %q, want %q", got, "Python")
	}
}

func TestTier(t *testing.T) {
	m := &python.Module{}
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want %d", got, 1)
	}
}

// --- Detection tests ---

func TestDetect_PyprojectTomlOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when pyproject.toml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "pyproject.toml")
	if result.SuggestedConfig.PackageManager != "pip" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "pip")
	}
}

func TestDetect_RequirementsTxtOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0.0\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when requirements.txt is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "requirements.txt")
}

func TestDetect_SetupPyOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "setup.py", "from setuptools import setup\nsetup(name='myapp')\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when setup.py is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "setup.py")
}

func TestDetect_PipfileOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Pipfile", "[packages]\nflask = \"*\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when Pipfile is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "Pipfile")
}

func TestDetect_UvLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "uv.lock", "# uv lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "uv" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "uv")
	}
	assertEvidenceContains(t, result.Evidence, "uv.lock")
}

func TestDetect_PoetryLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "poetry.lock", "# poetry lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "poetry" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "poetry")
	}
	assertEvidenceContains(t, result.Evidence, "poetry.lock")
}

func TestDetect_UvLockTakesPriorityOverPoetryLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "uv.lock", "# uv lockfile\n")
	writeFile(t, dir, "poetry.lock", "# poetry lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.PackageManager != "uv" {
		t.Errorf("PackageManager = %q, want %q (uv.lock should take priority over poetry.lock)",
			result.SuggestedConfig.PackageManager, "uv")
	}
}

func TestDetect_PythonVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, ".python-version", "3.11.5\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "3.11.5" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "3.11.5")
	}
	assertEvidenceContains(t, result.Evidence, ".python-version")
}

func TestDetect_PyprojectRequiresPython(t *testing.T) {
	dir := t.TempDir()
	pyproject := `[project]
name = "myapp"
requires-python = ">=3.10"
`
	writeFile(t, dir, "pyproject.toml", pyproject)

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "3.10" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "3.10")
	}
	assertEvidenceContains(t, result.Evidence, "requires-python")
}

func TestDetect_PythonVersionFileTakesPriorityOverRequiresPython(t *testing.T) {
	dir := t.TempDir()
	pyproject := `[project]
name = "myapp"
requires-python = ">=3.10"
`
	writeFile(t, dir, "pyproject.toml", pyproject)
	writeFile(t, dir, ".python-version", "3.13.0\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.Version != "3.13.0" {
		t.Errorf("Version = %q, want %q (.python-version should take priority)",
			result.SuggestedConfig.Version, "3.13.0")
	}
}

func TestDetect_RequiresPythonVariants(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		version string
	}{
		{"ge_only", `requires-python = ">=3.11"`, "3.11"},
		{"tilde_eq", `requires-python = "~=3.9"`, "3.9"},
		{"exact", `requires-python = "==3.12"`, "3.12"},
		{"lt", `requires-python = "<3.13"`, "3.13"},
		{"no_specifier", `requires-python = "3.10"`, "3.10"},
		{"with_spaces", `  requires-python = ">=3.11"`, "3.11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "pyproject.toml", "[project]\n"+tt.line+"\n")

			m := &python.Module{}
			result := m.Detect(dir)

			if result.SuggestedConfig.Version != tt.version {
				t.Errorf("Version = %q, want %q for line %q",
					result.SuggestedConfig.Version, tt.version, tt.line)
			}
		})
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &python.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDetect_HighestConfidenceWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0.0\n")
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain (pyproject.toml should elevate to Certain)",
			result.Confidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_Pip(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "pip",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "languages.python")
	assertContains(t, fragment, "enable = true")
	assertContains(t, fragment, `version = "3.12"`)
	assertContains(t, fragment, "venv.enable = true")
	assertNotContains(t, fragment, "uv.enable")
	assertNotContains(t, fragment, "poetry.enable")
}

func TestDevenvNixFragment_Uv(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "uv",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "uv.enable = true")
	assertNotContains(t, fragment, "poetry.enable")
}

func TestDevenvNixFragment_Poetry(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "poetry",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "poetry.enable = true")
	assertNotContains(t, fragment, "uv.enable")
}

func TestDevenvNixFragment_MutualExclusion(t *testing.T) {
	m := &python.Module{}

	// uv fragment should not contain poetry
	uvFrag, _ := m.DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: "uv"})
	if strings.Contains(uvFrag, "poetry.enable") {
		t.Error("uv fragment should not contain poetry.enable")
	}

	// poetry fragment should not contain uv
	poetryFrag, _ := m.DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: "poetry"})
	if strings.Contains(poetryFrag, "uv.enable") {
		t.Error("poetry fragment should not contain uv.enable")
	}
}

func TestDevenvNixFragment_DefaultVersion(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	assertContains(t, fragment, `version = "3.12"`)
}

func TestDevenvNixFragment_CustomVersion(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		Version: "3.11",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	assertContains(t, fragment, `version = "3.11"`)
	assertNotContains(t, fragment, `version = "3.12"`)
}

// --- DevenvYamlInputs tests ---

func TestDevenvYamlInputs(t *testing.T) {
	m := &python.Module{}
	inputs := m.DevenvYamlInputs(ecosystem.ModuleConfig{})
	if inputs != nil {
		t.Errorf("DevenvYamlInputs() = %v, want nil", inputs)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Pip(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pip"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(pip) returned %d files, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != "pip.conf" {
		t.Errorf("Path = %q, want %q", cfg.Path, "pip.conf")
	}

	content := string(cfg.Content)
	assertContains(t, content, "[global]")
	assertContains(t, content, "require-hashes = true")
	assertContains(t, content, "only-binary = :all:")
	assertContains(t, content, "Security-hardened pip configuration")
	assertContains(t, content, branding.GeneratedBy())
	assertContains(t, content, "pip >= 26.0")
}

func TestSecurityConfigs_PipDefault(t *testing.T) {
	m := &python.Module{}
	// Empty PackageManager should default to pip.
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(default) returned %d files, want 1", len(configs))
	}
	if configs[0].Path != "pip.conf" {
		t.Errorf("Path = %q, want %q", configs[0].Path, "pip.conf")
	}
}

func TestSecurityConfigs_Uv(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "uv"})

	if configs != nil {
		t.Errorf("SecurityConfigs(uv) = %v, want nil", configs)
	}
}

func TestSecurityConfigs_Poetry(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "poetry"})

	if configs != nil {
		t.Errorf("SecurityConfigs(poetry) = %v, want nil", configs)
	}
}

// --- Registry proxy tests ---

func TestSecurityConfigs_Pip_RegistryProxy(t *testing.T) {
	m := &python.Module{}
	proxy := "https://pypi.corp.example.com/simple/"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "pip",
		RegistryProxy:  proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(pip+proxy) returned %d files, want 1", len(configs))
	}

	content := string(configs[0].Content)
	assertContains(t, content, "index-url = "+proxy)
	// Existing security settings must be preserved.
	assertContains(t, content, "require-hashes = true")
	assertContains(t, content, "only-binary = :all:")
	assertContains(t, content, "[global]")
}

func TestSecurityConfigs_Pip_NoRegistryProxy(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pip"})

	content := string(configs[0].Content)
	assertNotContains(t, content, "index-url")
}

func TestSecurityConfigs_Pip_RegistryProxyPreservesExisting(t *testing.T) {
	m := &python.Module{}
	proxy := "https://pypi.corp.example.com/simple/"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "pip",
		RegistryProxy:  proxy,
	})

	content := string(configs[0].Content)
	assertContains(t, content, "Security-hardened pip configuration")
	assertContains(t, content, branding.GeneratedBy())
	assertContains(t, content, "require-hashes = true")
	assertContains(t, content, "only-binary = :all:")
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := &python.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 3 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 3", len(hooks))
	}

	expectedIDs := []string{"ruff", "mypy", "bandit"}

	for i, hook := range hooks {
		if hook.ID != expectedIDs[i] {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, expectedIDs[i])
		}
		if !hook.BuiltIn {
			t.Errorf("hooks[%d].BuiltIn = false, want true", i)
		}
		if hook.Language != "python" {
			t.Errorf("hooks[%d].Language = %q, want %q", i, hook.Language, "python")
		}
		if len(hook.Types) != 1 || hook.Types[0] != "python" {
			t.Errorf("hooks[%d].Types = %v, want [\"python\"]", i, hook.Types)
		}
		if hook.Name == "" {
			t.Errorf("hooks[%d].Name should not be empty", i)
		}
		if hook.Description == "" {
			t.Errorf("hooks[%d].Description should not be empty", i)
		}
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := &python.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	// Package install commands moved to base ask rules + package-guard hook.
	if len(rules) != 0 {
		t.Fatalf("DenyRules() returned %d rules, want 0 (installs handled by ask rules)", len(rules))
	}
}

// --- CICommands tests ---

func TestCICommands_Pip(t *testing.T) {
	m := &python.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "pip"})

	if len(cmds) != 3 {
		t.Fatalf("CICommands(pip) returned %d commands, want 3", len(cmds))
	}

	if cmds[0].Command != "pip install --require-hashes --only-binary :all: -r requirements.txt" {
		t.Errorf("install command = %q, want pip install with hashes", cmds[0].Command)
	}
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("install command Phase = %v, want CIPhaseInstall", cmds[0].Phase)
	}
	if cmds[1].Command != "pip-audit" {
		t.Errorf("scan command = %q, want pip-audit", cmds[1].Command)
	}
	if cmds[1].Phase != ecosystem.CIPhaseScan {
		t.Errorf("scan command Phase = %v, want CIPhaseScan", cmds[1].Phase)
	}
	if cmds[2].Command != "safety check" {
		t.Errorf("safety command = %q, want safety check", cmds[2].Command)
	}
}

func TestCICommands_Uv(t *testing.T) {
	m := &python.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "uv"})

	if len(cmds) != 3 {
		t.Fatalf("CICommands(uv) returned %d commands, want 3", len(cmds))
	}

	if cmds[0].Command != "uv sync --frozen --exclude-newer=7d" {
		t.Errorf("install command = %q, want 'uv sync --frozen --exclude-newer=7d'", cmds[0].Command)
	}
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("install command Phase = %v, want CIPhaseInstall", cmds[0].Phase)
	}
}

func TestCICommands_Poetry(t *testing.T) {
	m := &python.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "poetry"})

	if len(cmds) != 3 {
		t.Fatalf("CICommands(poetry) returned %d commands, want 3", len(cmds))
	}

	if cmds[0].Command != "poetry install --no-interaction" {
		t.Errorf("install command = %q, want 'poetry install --no-interaction'", cmds[0].Command)
	}
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("install command Phase = %v, want CIPhaseInstall", cmds[0].Phase)
	}
}

func TestCICommands_Default(t *testing.T) {
	m := &python.Module{}
	// Empty PackageManager should default to pip.
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 3 {
		t.Fatalf("CICommands(default) returned %d commands, want 3", len(cmds))
	}
	if cmds[0].Command != "pip install --require-hashes --only-binary :all: -r requirements.txt" {
		t.Errorf("default install command = %q, want pip install with hashes", cmds[0].Command)
	}
}

func TestCICommands_AllHaveAuditAndSafety(t *testing.T) {
	m := &python.Module{}
	for _, pm := range []string{"pip", "uv", "poetry"} {
		cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: pm})

		foundAudit := false
		foundSafety := false
		for _, cmd := range cmds {
			if cmd.Command == "pip-audit" {
				foundAudit = true
			}
			if cmd.Command == "safety check" {
				foundSafety = true
			}
		}
		if !foundAudit {
			t.Errorf("CICommands(%s) missing pip-audit command", pm)
		}
		if !foundSafety {
			t.Errorf("CICommands(%s) missing safety check command", pm)
		}
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := &python.Module{}
	pms := m.PackageManagers()

	if len(pms) != 3 {
		t.Fatalf("PackageManagers() returned %d entries, want 3", len(pms))
	}

	expectedNames := []string{"pip", "uv", "poetry"}
	for i, pm := range pms {
		if pm.Name != expectedNames[i] {
			t.Errorf("pms[%d].Name = %q, want %q", i, pm.Name, expectedNames[i])
		}
		if pm.LockFile == "" {
			t.Errorf("pms[%d].LockFile should not be empty", i)
		}
		if pm.FrozenInstallCommand == "" {
			t.Errorf("pms[%d].FrozenInstallCommand should not be empty", i)
		}
		if pm.AuditCommand == "" {
			t.Errorf("pms[%d].AuditCommand should not be empty", i)
		}
	}
}

func TestPackageManagers_AgeGating(t *testing.T) {
	m := &python.Module{}
	pms := m.PackageManagers()

	for _, pm := range pms {
		switch pm.Name {
		case "uv":
			if !pm.AgeGatingSupport {
				t.Errorf("uv should have AgeGatingSupport=true")
			}
		default:
			if pm.AgeGatingSupport {
				t.Errorf("%s should have AgeGatingSupport=false", pm.Name)
			}
		}
	}
}

func TestPackageManagers_LockFiles(t *testing.T) {
	m := &python.Module{}
	pms := m.PackageManagers()

	expectedLockFiles := map[string]string{
		"pip":    "requirements.txt",
		"uv":     "uv.lock",
		"poetry": "poetry.lock",
	}

	for _, pm := range pms {
		expected, ok := expectedLockFiles[pm.Name]
		if !ok {
			t.Errorf("unexpected package manager %q", pm.Name)
			continue
		}
		if pm.LockFile != expected {
			t.Errorf("pms[%s].LockFile = %q, want %q", pm.Name, pm.LockFile, expected)
		}
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := &python.Module{}
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("WizardFields() returned %d fields, want 2", len(fields))
	}

	// First field: package manager select
	pmField := fields[0]
	if pmField.Key != "python_package_manager" {
		t.Errorf("fields[0].Key = %q, want %q", pmField.Key, "python_package_manager")
	}
	if pmField.Type != ecosystem.FieldTypeSelect {
		t.Errorf("fields[0].Type = %v, want FieldTypeSelect", pmField.Type)
	}
	if len(pmField.Options) != 3 {
		t.Fatalf("fields[0].Options has %d entries, want 3", len(pmField.Options))
	}
	optionValues := make([]string, len(pmField.Options))
	for i, opt := range pmField.Options {
		optionValues[i] = opt.Value
	}
	expectedOptions := []string{"pip", "uv", "poetry"}
	for i, expected := range expectedOptions {
		if optionValues[i] != expected {
			t.Errorf("fields[0].Options[%d].Value = %q, want %q", i, optionValues[i], expected)
		}
	}

	// Second field: venv confirm
	venvField := fields[1]
	if venvField.Key != "python_venv" {
		t.Errorf("fields[1].Key = %q, want %q", venvField.Key, "python_venv")
	}
	if venvField.Type != ecosystem.FieldTypeConfirm {
		t.Errorf("fields[1].Type = %v, want FieldTypeConfirm", venvField.Type)
	}
	if venvField.Default != "true" {
		t.Errorf("fields[1].Default = %q, want %q", venvField.Default, "true")
	}
}

// --- Registration test ---

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("python")
	if !ok {
		t.Fatal("expected module 'python' to be registered in DefaultRegistry")
	}
	if mod.Name() != "python" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "python")
	}
}

// --- Test helpers ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected string NOT to contain %q\ngot:\n%s", substr, s)
	}
}

func assertEvidenceContains(t *testing.T, evidence []string, substr string) {
	t.Helper()
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("evidence %v should contain an entry mentioning %q", evidence, substr)
}
