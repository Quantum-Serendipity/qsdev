package docker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/docker"
)

// Compile-time interface compliance.
var _ ecosystem.EcosystemModule = (*docker.Module)(nil)

func newModule() *docker.Module {
	return &docker.Module{}
}

// ---------- Name / DisplayName / Tier ----------

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "docker" {
		t.Errorf("Name() = %q, want %q", got, "docker")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "Docker / Containerfiles" {
		t.Errorf("DisplayName() = %q, want %q", got, "Docker / Containerfiles")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want 1", got)
	}
}

// ---------- Detect ----------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writing fixture %s: %v", name, err)
	}
}

func TestDetect_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM alpine:3.19\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "Dockerfile found")
}

func TestDetect_Containerfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Containerfile", "FROM fedora:39\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "Containerfile found")
}

func TestDetect_DockerCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", "version: '3'\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "docker-compose.yml found")
	if result.SuggestedConfig.Extras["has_compose"] != "true" {
		t.Error("expected has_compose=true in Extras")
	}
}

func TestDetect_DockerComposeYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yaml", "version: '3'\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "docker-compose.yaml found")
	if result.SuggestedConfig.Extras["has_compose"] != "true" {
		t.Error("expected has_compose=true in Extras")
	}
}

func TestDetect_Dockerignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".dockerignore", "node_modules\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, ".dockerignore found")
}

func TestDetect_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM alpine\n")
	writeFile(t, dir, "Containerfile", "FROM fedora\n")
	writeFile(t, dir, "docker-compose.yml", "version: '3'\n")
	writeFile(t, dir, ".dockerignore", "*.tmp\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "Dockerfile found")
	assertEvidenceContains(t, result.Evidence, "Containerfile found")
	assertEvidenceContains(t, result.Evidence, "docker-compose.yml found")
	assertEvidenceContains(t, result.Evidence, ".dockerignore found")
	if result.SuggestedConfig.Extras["has_compose"] != "true" {
		t.Error("expected has_compose=true in Extras")
	}
}

func TestDetect_Empty(t *testing.T) {
	dir := t.TempDir()

	result := newModule().Detect(dir)
	if result.Detected {
		t.Fatal("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", result.Confidence)
	}
}

// ---------- DevenvNixFragment ----------

func TestDevenvNixFragment(t *testing.T) {
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	for _, pkg := range []string{"docker", "hadolint", "dive"} {
		if !strings.Contains(frag, pkg) {
			t.Errorf("fragment missing package %q", pkg)
		}
	}
	if !strings.Contains(frag, "packages = with pkgs;") {
		t.Error("fragment missing 'packages = with pkgs;' preamble")
	}
}

// ---------- SecurityConfigs ----------

func TestSecurityConfigs_Default(t *testing.T) {
	files := newModule().SecurityConfigs(ecosystem.ModuleConfig{})
	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}
	f := files[0]
	if f.Path != ".hadolint.yaml" {
		t.Errorf("Path = %q, want .hadolint.yaml", f.Path)
	}
	content := string(f.Content)

	for _, reg := range []string{"docker.io", "gcr.io", "ghcr.io"} {
		if !strings.Contains(content, reg) {
			t.Errorf("default config missing registry %q", reg)
		}
	}
	if !strings.Contains(content, "failure-threshold") {
		t.Error("missing failure-threshold key")
	}
	if !strings.Contains(content, "hadolint") {
		t.Error("missing hadolint reference in header comment")
	}
}

func TestSecurityConfigs_CustomRegistries(t *testing.T) {
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"trusted_registries": "my.registry.io, internal.corp",
		},
	}
	files := newModule().SecurityConfigs(cfg)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	content := string(files[0].Content)

	for _, reg := range []string{"my.registry.io", "internal.corp"} {
		if !strings.Contains(content, reg) {
			t.Errorf("custom config missing registry %q", reg)
		}
	}
	// Default registries should NOT be present.
	if strings.Contains(content, "docker.io") {
		t.Error("custom config unexpectedly contains default registry docker.io")
	}
}

func TestSecurityConfigs_ValidYAML(t *testing.T) {
	files := newModule().SecurityConfigs(ecosystem.ModuleConfig{})
	content := files[0].Content

	// Strip the comment header so YAML parses cleanly.
	var parsed struct {
		TrustedRegistries []string `yaml:"trustedRegistries"`
		FailureThreshold  string   `yaml:"failure-threshold"`
	}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v\ncontent:\n%s", err, content)
	}
	if len(parsed.TrustedRegistries) != 3 {
		t.Errorf("expected 3 trusted registries, got %d: %v", len(parsed.TrustedRegistries), parsed.TrustedRegistries)
	}
	if parsed.FailureThreshold != "warning" {
		t.Errorf("failure-threshold = %q, want %q", parsed.FailureThreshold, "warning")
	}
}

// ---------- PreCommitHooks ----------

func TestPreCommitHooks(t *testing.T) {
	hooks := newModule().PreCommitHooks(ecosystem.ModuleConfig{})
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
	h := hooks[0]
	if h.ID != "hadolint" {
		t.Errorf("hook ID = %q, want hadolint", h.ID)
	}
	if h.Language != "system" {
		t.Errorf("hook Language = %q, want system", h.Language)
	}
	if len(h.Types) != 1 || h.Types[0] != "dockerfile" {
		t.Errorf("hook Types = %v, want [dockerfile]", h.Types)
	}
	if !h.PassFilenames {
		t.Error("expected PassFilenames=true")
	}
	if h.Files != "(Dockerfile|Containerfile)" {
		t.Errorf("hook Files = %q, want (Dockerfile|Containerfile)", h.Files)
	}
	if h.BuiltIn {
		t.Error("expected BuiltIn=false")
	}
}

// ---------- DenyRules ----------

func TestDenyRules(t *testing.T) {
	rules := newModule().DenyRules(ecosystem.ModuleConfig{})
	if len(rules) != 1 {
		t.Fatalf("expected 1 deny rule, got %d", len(rules))
	}
	if rules[0] != "Bash(docker pull *)" {
		t.Errorf("deny rule = %q, want %q", rules[0], "Bash(docker pull *)")
	}
}

// ---------- CICommands ----------

func TestCICommands(t *testing.T) {
	cmds := newModule().CICommands(ecosystem.ModuleConfig{})
	if len(cmds) != 4 {
		t.Fatalf("expected 4 CI commands, got %d", len(cmds))
	}

	names := make(map[string]ecosystem.CICommand, len(cmds))
	for _, c := range cmds {
		names[c.Name] = c
	}

	// All should be Scan phase.
	for _, c := range cmds {
		if c.Phase != ecosystem.CIPhaseScan {
			t.Errorf("command %q phase = %v, want Scan", c.Name, c.Phase)
		}
	}

	if _, ok := names["hadolint"]; !ok {
		t.Error("missing hadolint CI command")
	}
	if _, ok := names["docker-build"]; !ok {
		t.Error("missing docker-build CI command")
	}
	if trivy, ok := names["trivy-image"]; !ok {
		t.Error("missing trivy-image CI command")
	} else if !strings.Contains(trivy.Description, "March 2026") {
		t.Error("trivy-image description should mention March 2026 compromise")
	}
	if _, ok := names["cosign-verify"]; !ok {
		t.Error("missing cosign-verify CI command")
	}
}

// ---------- PackageManagers ----------

func TestPackageManagers(t *testing.T) {
	pm := newModule().PackageManagers()
	if len(pm) != 0 {
		t.Errorf("expected nil/empty PackageManagers, got %d", len(pm))
	}
}

// ---------- WizardFields ----------

func TestWizardFields(t *testing.T) {
	fields := newModule().WizardFields()
	if len(fields) != 1 {
		t.Fatalf("expected 1 wizard field, got %d", len(fields))
	}
	f := fields[0]
	if f.Key != "trusted_registries" {
		t.Errorf("field Key = %q, want trusted_registries", f.Key)
	}
	if f.Type != ecosystem.FieldTypeInput {
		t.Errorf("field Type = %v, want FieldTypeInput", f.Type)
	}
	if f.Default != "docker.io,gcr.io,ghcr.io" {
		t.Errorf("field Default = %q, want docker.io,gcr.io,ghcr.io", f.Default)
	}
}

// ---------- helpers ----------

func assertEvidenceContains(t *testing.T, evidence []string, want string) {
	t.Helper()
	for _, e := range evidence {
		if e == want {
			return
		}
	}
	t.Errorf("evidence %v missing %q", evidence, want)
}
