package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Go ---

func TestDetectGo_WithPatchVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n\ngo 1.22.5\n")

	detected, version := detectGo(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if version != "1.22.5" {
		t.Errorf("version = %q, want %q", version, "1.22.5")
	}
}

func TestDetectGo_MinorOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n\ngo 1.22\n")

	detected, version := detectGo(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if version != "1.22" {
		t.Errorf("version = %q, want %q", version, "1.22")
	}
}

func TestDetectGo_NotPresent(t *testing.T) {
	dir := t.TempDir()

	detected, _ := detectGo(dir)
	if detected {
		t.Error("expected detected=false for empty dir")
	}
}

// --- Node ---

func TestDetectNode_NPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "package-lock.json"), "{}")

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "npm" {
		t.Errorf("pm = %q, want %q", pm, "npm")
	}
}

func TestDetectNode_Pnpm(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "lockfileVersion: '9.0'")

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "pnpm" {
		t.Errorf("pm = %q, want %q", pm, "pnpm")
	}
}

func TestDetectNode_Yarn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "yarn.lock"), "# yarn lockfile v1")

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "yarn" {
		t.Errorf("pm = %q, want %q", pm, "yarn")
	}
}

func TestDetectNode_Bun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "bun.lock"), "")

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "bun" {
		t.Errorf("pm = %q, want %q", pm, "bun")
	}
}

func TestDetectNode_BunBinary(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "bun.lockb"), "\x00binary")

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "bun" {
		t.Errorf("pm = %q, want %q", pm, "bun")
	}
}

func TestDetectNode_DefaultNPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

	detected, _, pm := detectNode(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "npm" {
		t.Errorf("pm = %q, want %q (default)", pm, "npm")
	}
}

func TestDetectNode_VersionFromNvmrc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, ".nvmrc"), "v22.1.0\n")

	_, version, _ := detectNode(dir)
	if version != "22.1.0" {
		t.Errorf("version = %q, want %q (v prefix stripped)", version, "22.1.0")
	}
}

func TestDetectNode_VersionFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test","engines":{"node":">=20.0.0"}}`)

	_, version, _ := detectNode(dir)
	if version != ">=20.0.0" {
		t.Errorf("version = %q, want %q", version, ">=20.0.0")
	}
}

func TestDetectNode_NotPresent(t *testing.T) {
	dir := t.TempDir()

	detected, _, _ := detectNode(dir)
	if detected {
		t.Error("expected detected=false for empty dir")
	}
}

// --- Python ---

func TestDetectPython_UV(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")
	writeFile(t, filepath.Join(dir, "uv.lock"), "")

	detected, _, pm := detectPython(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "uv" {
		t.Errorf("pm = %q, want %q", pm, "uv")
	}
}

func TestDetectPython_Poetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")
	writeFile(t, filepath.Join(dir, "poetry.lock"), "")

	detected, _, pm := detectPython(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "poetry" {
		t.Errorf("pm = %q, want %q", pm, "poetry")
	}
}

func TestDetectPython_Pip(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")
	writeFile(t, filepath.Join(dir, "requirements.txt"), "flask>=2.0")

	detected, _, pm := detectPython(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "pip" {
		t.Errorf("pm = %q, want %q", pm, "pip")
	}
}

func TestDetectPython_DefaultPip(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")

	detected, _, pm := detectPython(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if pm != "pip" {
		t.Errorf("pm = %q, want %q (default)", pm, "pip")
	}
}

func TestDetectPython_VersionFromPythonVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")
	writeFile(t, filepath.Join(dir, ".python-version"), "3.12.1\n")

	_, version, _ := detectPython(dir)
	if version != "3.12.1" {
		t.Errorf("version = %q, want %q", version, "3.12.1")
	}
}

func TestDetectPython_NotPresent(t *testing.T) {
	dir := t.TempDir()

	detected, _, _ := detectPython(dir)
	if detected {
		t.Error("expected detected=false for empty dir")
	}
}

// --- Rust ---

func TestDetectRust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Cargo.toml"), "[package]\nname = \"test\"")

	if !detectRust(dir) {
		t.Error("expected Rust detected")
	}
}

func TestDetectRust_NotPresent(t *testing.T) {
	dir := t.TempDir()
	if detectRust(dir) {
		t.Error("expected Rust not detected in empty dir")
	}
}

// --- Java ---

func TestDetectJava_Maven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pom.xml"), "<project></project>")

	hasMaven, hasGradle := detectJava(dir)
	if !hasMaven {
		t.Error("expected Maven detected")
	}
	if hasGradle {
		t.Error("expected Gradle not detected")
	}
}

func TestDetectJava_Gradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "build.gradle"), "plugins { id 'java' }")

	hasMaven, hasGradle := detectJava(dir)
	if hasMaven {
		t.Error("expected Maven not detected")
	}
	if !hasGradle {
		t.Error("expected Gradle detected")
	}
}

func TestDetectJava_GradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "build.gradle.kts"), "plugins { java }")

	_, hasGradle := detectJava(dir)
	if !hasGradle {
		t.Error("expected Gradle KTS detected")
	}
}

func TestDetectJava_NotPresent(t *testing.T) {
	dir := t.TempDir()
	hasMaven, hasGradle := detectJava(dir)
	if hasMaven || hasGradle {
		t.Error("expected Java not detected in empty dir")
	}
}

// --- .NET ---

func TestDetectDotNet_Csproj(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "test.csproj"), "<Project></Project>")

	if !detectDotNet(dir) {
		t.Error("expected .NET detected from .csproj")
	}
}

func TestDetectDotNet_Sln(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "test.sln"), "Microsoft Visual Studio Solution File")

	if !detectDotNet(dir) {
		t.Error("expected .NET detected from .sln")
	}
}

func TestDetectDotNet_NotPresent(t *testing.T) {
	dir := t.TempDir()
	if detectDotNet(dir) {
		t.Error("expected .NET not detected in empty dir")
	}
}

// --- Docker ---

func TestDetectDocker_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine")

	if !detectDocker(dir) {
		t.Error("expected Docker detected from Dockerfile")
	}
}

func TestDetectDocker_ComposeYml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "version: '3'")

	if !detectDocker(dir) {
		t.Error("expected Docker detected from docker-compose.yml")
	}
}

func TestDetectDocker_ComposeYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yaml"), "version: '3'")

	if !detectDocker(dir) {
		t.Error("expected Docker detected from docker-compose.yaml")
	}
}

func TestDetectDocker_NotPresent(t *testing.T) {
	dir := t.TempDir()
	if detectDocker(dir) {
		t.Error("expected Docker not detected in empty dir")
	}
}

// --- Terraform ---

func TestDetectTerraform(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `resource "null_resource" "test" {}`)

	if !detectTerraform(dir) {
		t.Error("expected Terraform detected from main.tf")
	}
}

func TestDetectTerraform_NotPresent(t *testing.T) {
	dir := t.TempDir()
	if detectTerraform(dir) {
		t.Error("expected Terraform not detected in empty dir")
	}
}

// --- Node version edge cases ---

func TestDetectNode_NvmrcWithoutPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, ".nvmrc"), "20.0.0")

	_, version, _ := detectNode(dir)
	if version != "20.0.0" {
		t.Errorf("version = %q, want %q", version, "20.0.0")
	}
}

func TestDetectNode_NvmrcOverridesPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test","engines":{"node":">=18"}}`)
	writeFile(t, filepath.Join(dir, ".nvmrc"), "v22.1.0")

	_, version, _ := detectNode(dir)
	if version != "22.1.0" {
		t.Errorf("version = %q, want %q (.nvmrc should take priority)", version, "22.1.0")
	}
}

// --- Java both Maven and Gradle ---

func TestDetectJava_Both(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pom.xml"), "<project></project>")
	writeFile(t, filepath.Join(dir, "build.gradle"), "")

	hasMaven, hasGradle := detectJava(dir)
	if !hasMaven || !hasGradle {
		t.Errorf("expected both Maven and Gradle detected, got maven=%v gradle=%v", hasMaven, hasGradle)
	}
}

// --- Multiple .NET files ---

func TestDetectDotNet_MultipleCsproj(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "A.csproj"), "<Project/>")
	writeFile(t, filepath.Join(dir, "B.csproj"), "<Project/>")

	if !detectDotNet(dir) {
		t.Error("expected .NET detected")
	}
}

// --- Multiple Terraform files ---

func TestDetectTerraform_Multiple(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), "")
	writeFile(t, filepath.Join(dir, "variables.tf"), "")

	if !detectTerraform(dir) {
		t.Error("expected Terraform detected")
	}
}

// --- Go version edge cases ---

func TestDetectGo_NoVersionLine(t *testing.T) {
	dir := t.TempDir()
	// A go.mod without a go directive.
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n")

	detected, version := detectGo(dir)
	if !detected {
		t.Fatal("expected detected=true even without version")
	}
	if version != "" {
		t.Errorf("version = %q, want empty", version)
	}
}

func TestDetectGo_ToolchainLine(t *testing.T) {
	// Ensure the "toolchain" line doesn't match.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n\ngo 1.23.4\n\ntoolchain go1.23.4\n")

	detected, version := detectGo(dir)
	if !detected {
		t.Fatal("expected detected=true")
	}
	if version != "1.23.4" {
		t.Errorf("version = %q, want %q", version, "1.23.4")
	}
}

// --- Python with both uv.lock and poetry.lock (uv wins) ---

func TestDetectPython_UvTakesPriority(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"test\"")
	writeFile(t, filepath.Join(dir, "uv.lock"), "")
	writeFile(t, filepath.Join(dir, "poetry.lock"), "")

	_, _, pm := detectPython(dir)
	if pm != "uv" {
		t.Errorf("pm = %q, want %q (uv.lock should take priority)", pm, "uv")
	}
}

func TestDetectDocker_BothDockerfileAndCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine")
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "version: '3'")

	if !detectDocker(dir) {
		t.Error("expected Docker detected")
	}
}

// --- Git remote with no origin ---

func TestDetectEnvironment_GitNoOrigin(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"), `[core]
	repositoryformatversion = 0
[remote "upstream"]
	url = https://github.com/upstream/repo.git
`)

	env := detectEnvironment(dir)
	if !env.IsGitRepo {
		t.Error("expected IsGitRepo to be true")
	}
	if env.RemoteURL != "" {
		t.Errorf("RemoteURL = %q, want empty (no origin remote)", env.RemoteURL)
	}
}
