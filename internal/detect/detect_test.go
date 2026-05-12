package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_GoProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/app\n\ngo 1.22.5\n")

	// Add a git repo.
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"), `[remote "origin"]
	url = git@github.com:example/app.git
`)

	dp := Detect(dir)

	if !dp.HasGoMod {
		t.Error("expected HasGoMod=true")
	}
	if dp.GoVersion != "1.22.5" {
		t.Errorf("GoVersion = %q, want %q", dp.GoVersion, "1.22.5")
	}
	if !dp.Ecosystems["go"] {
		t.Error("expected Ecosystems[go]=true")
	}
	if !dp.IsGitRepo {
		t.Error("expected IsGitRepo=true")
	}
	if dp.RemoteURL != "git@github.com:example/app.git" {
		t.Errorf("RemoteURL = %q, want git@github.com:example/app.git", dp.RemoteURL)
	}
	// Negative checks.
	if dp.HasPackageJSON {
		t.Error("expected HasPackageJSON=false")
	}
	if dp.HasCargoToml {
		t.Error("expected HasCargoToml=false")
	}
}

func TestDetect_NodeTypeScriptProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{
		"name": "my-app",
		"engines": {"node": ">=20"}
	}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "lockfileVersion: '9.0'")
	writeFile(t, filepath.Join(dir, "tsconfig.json"), "{}")

	dp := Detect(dir)

	if !dp.HasPackageJSON {
		t.Error("expected HasPackageJSON=true")
	}
	if dp.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q", dp.PackageManager, "pnpm")
	}
	if dp.NodeVersion != ">=20" {
		t.Errorf("NodeVersion = %q, want %q", dp.NodeVersion, ">=20")
	}
	if !dp.Ecosystems["node"] {
		t.Error("expected Ecosystems[node]=true")
	}
}

func TestDetect_MultiLanguageProject(t *testing.T) {
	dir := t.TempDir()

	// Go
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/multi\n\ngo 1.23\n")

	// Node
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"multi"}`)
	writeFile(t, filepath.Join(dir, "yarn.lock"), "# yarn")

	// Python
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"multi\"")
	writeFile(t, filepath.Join(dir, ".python-version"), "3.12.1")
	writeFile(t, filepath.Join(dir, "poetry.lock"), "")

	// Rust
	writeFile(t, filepath.Join(dir, "Cargo.toml"), "[package]\nname = \"multi\"")

	// Docker
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine")

	// Terraform
	writeFile(t, filepath.Join(dir, "main.tf"), "")

	// Environment
	writeFile(t, filepath.Join(dir, "devenv.nix"), "{}")
	writeFile(t, filepath.Join(dir, ".envrc"), "use devenv")

	dp := Detect(dir)

	// Languages
	if !dp.HasGoMod || dp.GoVersion != "1.23" {
		t.Errorf("Go: HasGoMod=%v, GoVersion=%q", dp.HasGoMod, dp.GoVersion)
	}
	if !dp.HasPackageJSON || dp.PackageManager != "yarn" {
		t.Errorf("Node: HasPackageJSON=%v, PM=%q", dp.HasPackageJSON, dp.PackageManager)
	}
	if !dp.HasPyProject || dp.PythonVersion != "3.12.1" {
		t.Errorf("Python: HasPyProject=%v, Version=%q", dp.HasPyProject, dp.PythonVersion)
	}
	if !dp.HasCargoToml {
		t.Error("expected HasCargoToml=true")
	}
	if !dp.HasDockerfile {
		t.Error("expected HasDockerfile=true")
	}
	if !dp.HasTerraform {
		t.Error("expected HasTerraform=true")
	}

	// Ecosystems map
	expectedEcosystems := []string{"go", "node", "python", "rust", "docker", "terraform"}
	for _, eco := range expectedEcosystems {
		if !dp.Ecosystems[eco] {
			t.Errorf("expected Ecosystems[%q]=true", eco)
		}
	}

	// Environment
	if !dp.HasDevenvNix {
		t.Error("expected HasDevenvNix=true")
	}
	if !dp.HasEnvrc {
		t.Error("expected HasEnvrc=true")
	}
}

func TestDetect_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	dp := Detect(dir)

	if dp.HasGoMod || dp.HasPackageJSON || dp.HasCargoToml || dp.HasPyProject ||
		dp.HasPomXML || dp.HasBuildGradle || dp.HasCsproj || dp.HasDockerfile ||
		dp.HasTerraform {
		t.Error("expected no languages detected in empty directory")
	}
	if dp.HasDevenvNix || dp.HasDevenvYaml || dp.HasClaudeDir || dp.HasClaudeMd ||
		dp.HasClaudeSettings || dp.HasEnvrc || dp.HasMcpJson {
		t.Error("expected no environment state in empty directory")
	}
	if dp.IsGitRepo || dp.HasGitHooks || dp.RemoteURL != "" {
		t.Error("expected no git state in empty directory")
	}
	if len(dp.Ecosystems) != 0 {
		t.Errorf("expected empty Ecosystems map, got %v", dp.Ecosystems)
	}
}

func TestDetect_JavaProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pom.xml"), "<project></project>")
	writeFile(t, filepath.Join(dir, "build.gradle.kts"), "plugins { java }")

	dp := Detect(dir)

	if !dp.HasPomXML {
		t.Error("expected HasPomXML=true")
	}
	if !dp.HasBuildGradle {
		t.Error("expected HasBuildGradle=true")
	}
	if !dp.Ecosystems["java"] {
		t.Error("expected Ecosystems[java]=true")
	}
}

func TestDetect_DotNetProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "MyApp.csproj"), "<Project/>")

	dp := Detect(dir)

	if !dp.HasCsproj {
		t.Error("expected HasCsproj=true")
	}
	if !dp.Ecosystems["dotnet"] {
		t.Error("expected Ecosystems[dotnet]=true")
	}
}

func TestDetect_FullEnvironmentState(t *testing.T) {
	dir := t.TempDir()

	// Set up all environment files.
	writeFile(t, filepath.Join(dir, "devenv.nix"), "{}")
	writeFile(t, filepath.Join(dir, "devenv.yaml"), "inputs: {}")
	if err := os.Mkdir(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, ".claude", "settings.json"), "{}")
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# Test")
	writeFile(t, filepath.Join(dir, ".envrc"), "use devenv")
	writeFile(t, filepath.Join(dir, ".mcp.json"), "{}")

	// Git repo with hooks.
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"), `[remote "origin"]
	url = https://github.com/test/repo.git
`)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	dp := Detect(dir)

	if !dp.HasDevenvNix {
		t.Error("expected HasDevenvNix=true")
	}
	if !dp.HasDevenvYaml {
		t.Error("expected HasDevenvYaml=true")
	}
	if !dp.HasClaudeDir {
		t.Error("expected HasClaudeDir=true")
	}
	if !dp.HasClaudeMd {
		t.Error("expected HasClaudeMd=true")
	}
	if !dp.HasClaudeSettings {
		t.Error("expected HasClaudeSettings=true")
	}
	if !dp.HasEnvrc {
		t.Error("expected HasEnvrc=true")
	}
	if !dp.HasMcpJson {
		t.Error("expected HasMcpJson=true")
	}
	if !dp.IsGitRepo {
		t.Error("expected IsGitRepo=true")
	}
	if !dp.HasGitHooks {
		t.Error("expected HasGitHooks=true")
	}
	if dp.RemoteURL != "https://github.com/test/repo.git" {
		t.Errorf("RemoteURL = %q, want https://github.com/test/repo.git", dp.RemoteURL)
	}
}

func BenchmarkDetect(b *testing.B) {
	dir := b.TempDir()

	// Set up a realistic multi-language project.
	writeFileB(b, filepath.Join(dir, "go.mod"), "module example.com/bench\n\ngo 1.22.5\n")
	writeFileB(b, filepath.Join(dir, "package.json"), `{"name":"bench","engines":{"node":">=20"}}`)
	writeFileB(b, filepath.Join(dir, "package-lock.json"), "{}")
	writeFileB(b, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"bench\"")
	writeFileB(b, filepath.Join(dir, ".python-version"), "3.12.1")
	writeFileB(b, filepath.Join(dir, "Cargo.toml"), "[package]\nname = \"bench\"")
	writeFileB(b, filepath.Join(dir, "Dockerfile"), "FROM alpine")
	writeFileB(b, filepath.Join(dir, "main.tf"), "")
	writeFileB(b, filepath.Join(dir, "devenv.nix"), "{}")
	writeFileB(b, filepath.Join(dir, ".envrc"), "use devenv")

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0o755); err != nil {
		b.Fatal(err)
	}
	writeFileB(b, filepath.Join(gitDir, "config"), `[remote "origin"]
	url = git@github.com:example/bench.git
`)

	b.ResetTimer()
	for range b.N {
		dp := Detect(dir)
		if !dp.HasGoMod {
			b.Fatal("detection failed during benchmark")
		}
	}
}

func writeFileB(b *testing.B, path, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
}
