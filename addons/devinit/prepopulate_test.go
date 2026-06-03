package devinit_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestMapDetectionToDefaults_Go(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:  true,
		GoVersion: "1.24",
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/myproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "go" {
		t.Errorf("expected language name %q, got %q", "go", answers.Languages[0].Name)
	}
	if answers.Languages[0].Version != "1.24" {
		t.Errorf("expected version %q, got %q", "1.24", answers.Languages[0].Version)
	}
}

func TestMapDetectionToDefaults_JavaScript_NotNode(t *testing.T) {
	// CRITICAL: detection engine uses "node" internally but the canonical
	// language name must be "javascript".
	detected := types.DetectedProject{
		HasPackageJSON: true,
		NodeVersion:    "22",
		PackageManager: "pnpm",
		Ecosystems:     map[string]bool{"node": true},
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/jsproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	lang := answers.Languages[0]
	if lang.Name != "javascript" {
		t.Errorf("expected language name %q, got %q (must NOT be 'node')", "javascript", lang.Name)
	}
	if lang.Version != "22" {
		t.Errorf("expected version %q, got %q", "22", lang.Version)
	}
	if lang.PackageManager != "pnpm" {
		t.Errorf("expected package manager %q, got %q", "pnpm", lang.PackageManager)
	}
}

func TestMapDetectionToDefaults_Python(t *testing.T) {
	detected := types.DetectedProject{
		HasPyProject:  true,
		PythonVersion: "3.12",
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/pyproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "python" {
		t.Errorf("expected language name %q, got %q", "python", answers.Languages[0].Name)
	}
	if answers.Languages[0].Version != "3.12" {
		t.Errorf("expected version %q, got %q", "3.12", answers.Languages[0].Version)
	}
}

func TestMapDetectionToDefaults_Rust(t *testing.T) {
	detected := types.DetectedProject{HasCargoToml: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/rustproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "rust" {
		t.Errorf("expected language name %q, got %q", "rust", answers.Languages[0].Name)
	}
}

func TestMapDetectionToDefaults_JavaMavenOnly(t *testing.T) {
	detected := types.DetectedProject{HasPomXML: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/javaproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	lang := answers.Languages[0]
	if lang.Name != "java" {
		t.Errorf("expected language name %q, got %q", "java", lang.Name)
	}
	if len(lang.Extras) != 1 || lang.Extras[0] != "build_tool=maven" {
		t.Errorf("expected extras [build_tool=maven], got %v", lang.Extras)
	}
}

func TestMapDetectionToDefaults_JavaGradleOnly(t *testing.T) {
	detected := types.DetectedProject{HasBuildGradle: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/javaproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	lang := answers.Languages[0]
	if lang.Name != "java" {
		t.Errorf("expected language name %q, got %q", "java", lang.Name)
	}
	if len(lang.Extras) != 1 || lang.Extras[0] != "build_tool=gradle" {
		t.Errorf("expected extras [build_tool=gradle], got %v", lang.Extras)
	}
}

func TestMapDetectionToDefaults_JavaBoth(t *testing.T) {
	detected := types.DetectedProject{HasPomXML: true, HasBuildGradle: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/javaproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	lang := answers.Languages[0]
	if lang.Name != "java" {
		t.Errorf("expected language name %q, got %q", "java", lang.Name)
	}
	if len(lang.Extras) != 1 || lang.Extras[0] != "build_tool=both" {
		t.Errorf("expected extras [build_tool=both], got %v", lang.Extras)
	}
}

func TestMapDetectionToDefaults_DotNet(t *testing.T) {
	detected := types.DetectedProject{HasCsproj: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/dotnetproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "dotnet" {
		t.Errorf("expected language name %q, got %q", "dotnet", answers.Languages[0].Name)
	}
}

func TestMapDetectionToDefaults_Container(t *testing.T) {
	detected := types.DetectedProject{HasDockerfile: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/containerproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "container" {
		t.Errorf("expected language name %q, got %q", "container", answers.Languages[0].Name)
	}
}

func TestMapDetectionToDefaults_ContainerWithRuntimeAndOSFamily(t *testing.T) {
	detected := types.DetectedProject{
		HasDockerfile:    true,
		ContainerRuntime: "podman-rootless",
		OSFamily:         "nixos",
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/podmanproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	lc := answers.Languages[0]
	hasRuntime := false
	hasOSFamily := false
	for _, e := range lc.Extras {
		if e == "container_runtime=podman-rootless" {
			hasRuntime = true
		}
		if e == "os_family=nixos" {
			hasOSFamily = true
		}
	}
	if !hasRuntime {
		t.Errorf("Extras missing container_runtime, got %v", lc.Extras)
	}
	if !hasOSFamily {
		t.Errorf("Extras missing os_family, got %v", lc.Extras)
	}
}

func TestMapDetectionToDefaults_Terraform(t *testing.T) {
	detected := types.DetectedProject{HasTerraform: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/tfproject")

	if len(answers.Languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "terraform" {
		t.Errorf("expected language name %q, got %q", "terraform", answers.Languages[0].Name)
	}
}

func TestMapDetectionToDefaults_MultiLanguage(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:       true,
		GoVersion:      "1.24",
		HasPackageJSON: true,
		NodeVersion:    "22",
		PackageManager: "npm",
		HasDockerfile:  true,
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/multi")

	if len(answers.Languages) != 3 {
		t.Fatalf("expected 3 languages, got %d", len(answers.Languages))
	}

	names := make(map[string]bool)
	for _, lang := range answers.Languages {
		names[lang.Name] = true
	}
	for _, expected := range []string{"go", "javascript", "container"} {
		if !names[expected] {
			t.Errorf("expected language %q in results", expected)
		}
	}
}

func TestMapDetectionToDefaults_EmptyDetection(t *testing.T) {
	detected := types.DetectedProject{}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/empty")

	if len(answers.Languages) != 0 {
		t.Errorf("expected 0 languages for empty detection, got %d", len(answers.Languages))
	}
	if answers.ProjectName != "empty" {
		t.Errorf("expected project name %q, got %q", "empty", answers.ProjectName)
	}
	if answers.Direnv {
		t.Error("Direnv should be false for empty detection")
	}
	if answers.ClaudeCode {
		t.Error("ClaudeCode should be false for empty detection")
	}
}

func TestMapDetectionToDefaults_Direnv(t *testing.T) {
	detected := types.DetectedProject{HasEnvrc: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	if !answers.Direnv {
		t.Error("expected Direnv to be true when .envrc detected")
	}
}

func TestMapDetectionToDefaults_ClaudeCode_Dir(t *testing.T) {
	detected := types.DetectedProject{HasClaudeDir: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode to be true when .claude/ detected")
	}
}

func TestMapDetectionToDefaults_ClaudeCode_Md(t *testing.T) {
	detected := types.DetectedProject{HasClaudeMd: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode to be true when CLAUDE.md detected")
	}
}

func TestMapDetectionToDefaults_ClaudeCode_Settings(t *testing.T) {
	detected := types.DetectedProject{HasClaudeSettings: true}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode to be true when claude settings detected")
	}
}

func TestMapDetectionToDefaults_DetectedAndProjectRoot(t *testing.T) {
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/myrepo")

	if answers.ProjectRoot != "/tmp/myrepo" {
		t.Errorf("expected project root %q, got %q", "/tmp/myrepo", answers.ProjectRoot)
	}
	if !answers.Detected.HasGoMod {
		t.Error("expected Detected.HasGoMod to be true")
	}
}

func TestProjectName_FromHTTPSRemoteURL(t *testing.T) {
	detected := types.DetectedProject{
		RemoteURL: "https://github.com/myorg/myrepo.git",
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/fallback")

	if answers.ProjectName != "myrepo" {
		t.Errorf("expected project name %q, got %q", "myrepo", answers.ProjectName)
	}
}

func TestProjectName_FromSSHRemoteURL(t *testing.T) {
	detected := types.DetectedProject{
		RemoteURL: "git@github.com:myorg/myrepo.git",
	}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/tmp/fallback")

	if answers.ProjectName != "myrepo" {
		t.Errorf("expected project name %q, got %q", "myrepo", answers.ProjectName)
	}
}

func TestProjectName_FallbackToDirectory(t *testing.T) {
	detected := types.DetectedProject{}
	answers := devinit.ExportMapDetectionToDefaults(detected, "/home/user/projects/coolapp")

	if answers.ProjectName != "coolapp" {
		t.Errorf("expected project name %q, got %q", "coolapp", answers.ProjectName)
	}
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"HTTPS with .git", "https://github.com/org/repo.git", "repo"},
		{"HTTPS without .git", "https://github.com/org/repo", "repo"},
		{"SSH with .git", "git@github.com:org/repo.git", "repo"},
		{"SSH without .git", "git@github.com:org/repo", "repo"},
		{"HTTPS trailing slash", "https://github.com/org/repo/", "repo"},
		{"HTTPS deep path", "https://gitlab.com/group/subgroup/repo.git", "repo"},
		{"SSH deep path", "git@gitlab.com:group/subgroup/repo.git", "repo"},
		{"Empty string", "", ""},
		{"Just a name", "myrepo", "myrepo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := devinit.ExportExtractRepoName(tt.url)
			if got != tt.want {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
