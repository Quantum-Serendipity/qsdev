package container_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/container"
)

// Compile-time interface compliance.
var _ ecosystem.EcosystemModule = (*container.Module)(nil)
var _ ecosystem.PackageProvider = (*container.Module)(nil)

func newModule() *container.Module {
	return &container.Module{}
}

// ---------- Name / DisplayName / Tier ----------

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "container", "Containers", 1)
}

// ---------- Detect ----------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
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

// ---------- DevenvPackages ----------

func TestDevenvPackages_Docker(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "docker"},
	}
	pkgs := newModule().DevenvPackages(cfg)
	want := []string{"docker", "hadolint", "dive"}
	if len(pkgs) != len(want) {
		t.Fatalf("DevenvPackages(docker) = %v, want %v", pkgs, want)
	}
	for i, w := range want {
		if pkgs[i] != w {
			t.Errorf("DevenvPackages(docker)[%d] = %q, want %q", i, pkgs[i], w)
		}
	}
}

func TestDevenvPackages_Podman(t *testing.T) {
	t.Parallel()
	for _, rt := range []string{"podman-rootless", "podman-rootful"} {
		t.Run(rt, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{
				Extras: map[string]string{"container_runtime": rt},
			}
			pkgs := newModule().DevenvPackages(cfg)
			want := []string{"podman", "podman-compose", "buildah", "skopeo", "hadolint", "dive"}
			if len(pkgs) != len(want) {
				t.Fatalf("DevenvPackages(%s) = %v, want %v", rt, pkgs, want)
			}
			for i, w := range want {
				if pkgs[i] != w {
					t.Errorf("DevenvPackages(%s)[%d] = %q, want %q", rt, i, pkgs[i], w)
				}
			}
		})
	}
}

func TestDevenvPackages_NoRuntime(t *testing.T) {
	t.Parallel()
	pkgs := newModule().DevenvPackages(ecosystem.ModuleConfig{})
	// Default should be docker packages.
	want := []string{"docker", "hadolint", "dive"}
	if len(pkgs) != len(want) {
		t.Fatalf("DevenvPackages(default) = %v, want %v", pkgs, want)
	}
	for i, w := range want {
		if pkgs[i] != w {
			t.Errorf("DevenvPackages(default)[%d] = %q, want %q", i, pkgs[i], w)
		}
	}
}

// ---------- DevenvNixFragment ----------

func TestDevenvNixFragment(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	// Default (docker) fragment should be empty — packages are via DevenvPackages.
	if frag != "" {
		t.Errorf("default fragment should be empty, got %q", frag)
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

// ---------- CICommands ----------

func TestCICommands(t *testing.T) {
	t.Parallel()
	cmds := newModule().CICommands(ecosystem.ModuleConfig{})
	if len(cmds) != 5 {
		t.Fatalf("expected 5 CI commands, got %d", len(cmds))
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
	if _, ok := names["container-build"]; !ok {
		t.Error("missing container-build CI command")
	}
	if _, ok := names["syft-sbom"]; !ok {
		t.Error("missing syft-sbom CI command")
	}
	if _, ok := names["grype-scan"]; !ok {
		t.Error("missing grype-scan CI command")
	}
	if _, ok := names["cosign-verify"]; !ok {
		t.Error("missing cosign-verify CI command")
	}

	// trivy-image should be absent (replaced by syft-sbom + grype-scan).
	if _, ok := names["trivy-image"]; ok {
		t.Error("trivy-image should be absent — replaced by syft-sbom + grype-scan")
	}
}

func TestCICommands_DockerRuntime(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "docker"},
	}
	cmds := newModule().CICommands(cfg)

	for _, c := range cmds {
		if strings.Contains(c.Command, "podman") {
			t.Errorf("command %q should use docker, not podman: %s", c.Name, c.Command)
		}
	}

	names := make(map[string]ecosystem.CICommand, len(cmds))
	for _, c := range cmds {
		names[c.Name] = c
	}
	if build, ok := names["container-build"]; ok {
		if !strings.HasPrefix(build.Command, "docker ") {
			t.Errorf("container-build should start with 'docker ', got %q", build.Command)
		}
	}
}

func TestCICommands_PodmanRuntime(t *testing.T) {
	t.Parallel()
	for _, rt := range []string{"podman-rootless", "podman-rootful"} {
		t.Run(rt, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{
				Extras: map[string]string{"container_runtime": rt},
			}
			cmds := newModule().CICommands(cfg)

			names := make(map[string]ecosystem.CICommand, len(cmds))
			for _, c := range cmds {
				names[c.Name] = c
			}

			if build, ok := names["container-build"]; ok {
				if !strings.HasPrefix(build.Command, "podman ") {
					t.Errorf("container-build should start with 'podman ', got %q", build.Command)
				}
			}
			if sbom, ok := names["syft-sbom"]; ok {
				if !strings.Contains(sbom.Command, "podman images") {
					t.Errorf("syft-sbom should use 'podman images', got %q", sbom.Command)
				}
			}
			if cosign, ok := names["cosign-verify"]; ok {
				if !strings.Contains(cosign.Command, "podman images") {
					t.Errorf("cosign-verify should use 'podman images', got %q", cosign.Command)
				}
			}
		})
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

// ---------- DevenvNixFragment (runtime-aware) ----------

func TestDevenvNixFragment_DockerRuntime(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "docker"},
	}
	frag, err := newModule().DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	// Docker fragment should be empty — packages provided via DevenvPackages.
	if frag != "" {
		t.Errorf("Docker runtime fragment should be empty, got %q", frag)
	}
}

func TestDevenvNixFragment_PodmanRuntime(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "podman-rootless"},
	}
	frag, err := newModule().DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if !strings.Contains(frag, "DOCKER_HOST") {
		t.Error("Podman fragment should set DOCKER_HOST env var")
	}
	// Fragment should NOT contain packages — those are in DevenvPackages.
	if strings.Contains(frag, "packages") {
		t.Error("Podman fragment should not contain packages block")
	}
}

func TestDevenvNixFragment_NoRuntime(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	// Default (docker) fragment should be empty.
	if frag != "" {
		t.Errorf("no-runtime fragment should be empty, got %q", frag)
	}
}

// ---------- DevenvYamlInputs (runtime-aware) ----------

func TestDevenvYamlInputs_PodmanNixOS(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"container_runtime": "podman-rootless",
			"os_family":         "nixos",
		},
	}
	inputs := newModule().DevenvYamlInputs(cfg)
	if len(inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(inputs))
	}
	if !strings.Contains(inputs[0].URL, "quadlet-nix") {
		t.Errorf("input URL should reference quadlet-nix, got %q", inputs[0].URL)
	}
}

func TestDevenvYamlInputs_PodmanNonNixOS(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"container_runtime": "podman-rootless",
			"os_family":         "ubuntu",
		},
	}
	inputs := newModule().DevenvYamlInputs(cfg)
	if len(inputs) != 0 {
		t.Errorf("expected nil/empty inputs for non-NixOS Podman, got %d", len(inputs))
	}
}

func TestDevenvYamlInputs_Docker(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "docker"},
	}
	inputs := newModule().DevenvYamlInputs(cfg)
	if len(inputs) != 0 {
		t.Errorf("expected nil/empty inputs for Docker, got %d", len(inputs))
	}
}

// ---------- VerificationCommands (runtime-aware) ----------

func TestVerificationCommands_DockerRuntime(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "docker"},
	}
	vc := newModule().VerificationCommands(cfg)
	if len(vc.Build) != 1 || vc.Build[0] != "docker build ." {
		t.Errorf("Build = %v, want [docker build .]", vc.Build)
	}
}

func TestVerificationCommands_PodmanRuntime(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "podman-rootless"},
	}
	vc := newModule().VerificationCommands(cfg)
	if len(vc.Build) != 1 || vc.Build[0] != "podman build ." {
		t.Errorf("Build = %v, want [podman build .]", vc.Build)
	}
}

// ---------- DenyRules (runtime-aware) ----------

// TestDenyRules_BlocksEscapesForEveryRuntime runs representative commands
// through the project's deny matcher. Container-escape arguments and image
// pulls must be denied for both CLIs whatever runtime is configured, while
// ordinary build/run commands stay allowed.
func TestDenyRules_BlocksEscapesForEveryRuntime(t *testing.T) {
	t.Parallel()

	denied := []string{
		"docker pull evil/image",
		"docker pull",
		"podman pull evil/image",
		"docker run --privileged -v /:/host alpine chroot /host",
		"podman run --privileged alpine",
		"docker run -v /var/run/docker.sock:/var/run/docker.sock alpine",
		"podman run -v /run/podman/podman.sock:/sock alpine",
		"docker run --rm -v /:/host alpine",
		"docker run --volume=/:/host alpine",
		"docker run -v/:/host alpine",
		"docker run --mount type=bind,source=/,target=/host alpine",
		"podman run --mount type=bind,src=/,dst=/host alpine",
		"docker run --pid=host alpine",
		"podman run --network=host alpine",
		"docker run --net host alpine",
		"docker exec --privileged ctr sh",
		"docker container run --privileged alpine",
	}
	allowed := []string{
		"docker build .",
		"podman build .",
		"docker run --rm -v ./src:/src alpine ls",
		"docker run --mount type=bind,source=/home/me/src,target=/src alpine",
		"docker images -q",
		"podman ps",
	}

	for _, rt := range []string{"", "docker", "podman-rootless", "podman-rootful"} {
		t.Run("runtime="+rt, func(t *testing.T) {
			t.Parallel()
			rules := newModule().DenyRules(ecosystem.ModuleConfig{
				Extras: map[string]string{"container_runtime": rt},
			})
			for _, cmd := range denied {
				if !deniedBy(rules, cmd) {
					t.Errorf("%q is not denied", cmd)
				}
			}
			for _, cmd := range allowed {
				if deniedBy(rules, cmd) {
					t.Errorf("%q is unexpectedly denied", cmd)
				}
			}
		})
	}
}

func deniedBy(rules []string, cmd string) bool {
	for _, r := range rules {
		if denyutil.MatchesDenyRule(r, "Bash("+cmd+")") {
			return true
		}
	}
	return false
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
