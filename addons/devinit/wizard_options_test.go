package devinit_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devinit"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestDetectionAnnotation_Go(t *testing.T) {
	detected := types.DetectedProject{HasGoMod: true}
	ann := devinit.ExportDetectionAnnotation("go", detected)
	if ann != "(detected: go.mod)" {
		t.Errorf("expected %q, got %q", "(detected: go.mod)", ann)
	}
}

func TestDetectionAnnotation_JavaScript(t *testing.T) {
	detected := types.DetectedProject{HasPackageJSON: true}
	ann := devinit.ExportDetectionAnnotation("javascript", detected)
	if ann != "(detected: package.json)" {
		t.Errorf("expected %q, got %q", "(detected: package.json)", ann)
	}
}

func TestDetectionAnnotation_JavaScriptWithPnpm(t *testing.T) {
	detected := types.DetectedProject{HasPackageJSON: true, PackageManager: "pnpm"}
	ann := devinit.ExportDetectionAnnotation("javascript", detected)
	if ann != "(detected: package.json + pnpm-lock.yaml)" {
		t.Errorf("expected %q, got %q", "(detected: package.json + pnpm-lock.yaml)", ann)
	}
}

func TestDetectionAnnotation_JavaScriptWithNpm(t *testing.T) {
	detected := types.DetectedProject{HasPackageJSON: true, PackageManager: "npm"}
	ann := devinit.ExportDetectionAnnotation("javascript", detected)
	if ann != "(detected: package.json + package-lock.json)" {
		t.Errorf("expected %q, got %q", "(detected: package.json + package-lock.json)", ann)
	}
}

func TestDetectionAnnotation_JavaScriptWithYarn(t *testing.T) {
	detected := types.DetectedProject{HasPackageJSON: true, PackageManager: "yarn"}
	ann := devinit.ExportDetectionAnnotation("javascript", detected)
	if ann != "(detected: package.json + yarn.lock)" {
		t.Errorf("expected %q, got %q", "(detected: package.json + yarn.lock)", ann)
	}
}

func TestDetectionAnnotation_JavaScriptWithBun(t *testing.T) {
	detected := types.DetectedProject{HasPackageJSON: true, PackageManager: "bun"}
	ann := devinit.ExportDetectionAnnotation("javascript", detected)
	if ann != "(detected: package.json + bun.lockb)" {
		t.Errorf("expected %q, got %q", "(detected: package.json + bun.lockb)", ann)
	}
}

func TestDetectionAnnotation_Python(t *testing.T) {
	detected := types.DetectedProject{HasPyProject: true}
	ann := devinit.ExportDetectionAnnotation("python", detected)
	if ann != "(detected: pyproject.toml)" {
		t.Errorf("expected %q, got %q", "(detected: pyproject.toml)", ann)
	}
}

func TestDetectionAnnotation_Rust(t *testing.T) {
	detected := types.DetectedProject{HasCargoToml: true}
	ann := devinit.ExportDetectionAnnotation("rust", detected)
	if ann != "(detected: Cargo.toml)" {
		t.Errorf("expected %q, got %q", "(detected: Cargo.toml)", ann)
	}
}

func TestDetectionAnnotation_JavaMaven(t *testing.T) {
	detected := types.DetectedProject{HasPomXML: true}
	ann := devinit.ExportDetectionAnnotation("java", detected)
	if ann != "(detected: pom.xml)" {
		t.Errorf("expected %q, got %q", "(detected: pom.xml)", ann)
	}
}

func TestDetectionAnnotation_JavaGradle(t *testing.T) {
	detected := types.DetectedProject{HasBuildGradle: true}
	ann := devinit.ExportDetectionAnnotation("java", detected)
	if ann != "(detected: build.gradle)" {
		t.Errorf("expected %q, got %q", "(detected: build.gradle)", ann)
	}
}

func TestDetectionAnnotation_JavaBoth(t *testing.T) {
	detected := types.DetectedProject{HasPomXML: true, HasBuildGradle: true}
	ann := devinit.ExportDetectionAnnotation("java", detected)
	if ann != "(detected: pom.xml + build.gradle)" {
		t.Errorf("expected %q, got %q", "(detected: pom.xml + build.gradle)", ann)
	}
}

func TestDetectionAnnotation_DotNet(t *testing.T) {
	detected := types.DetectedProject{HasCsproj: true}
	ann := devinit.ExportDetectionAnnotation("dotnet", detected)
	if ann != "(detected: *.csproj)" {
		t.Errorf("expected %q, got %q", "(detected: *.csproj)", ann)
	}
}

func TestDetectionAnnotation_Docker(t *testing.T) {
	detected := types.DetectedProject{HasDockerfile: true}
	ann := devinit.ExportDetectionAnnotation("docker", detected)
	if ann != "(detected: Dockerfile)" {
		t.Errorf("expected %q, got %q", "(detected: Dockerfile)", ann)
	}
}

func TestDetectionAnnotation_Terraform(t *testing.T) {
	detected := types.DetectedProject{HasTerraform: true}
	ann := devinit.ExportDetectionAnnotation("terraform", detected)
	if ann != "(detected: *.tf)" {
		t.Errorf("expected %q, got %q", "(detected: *.tf)", ann)
	}
}

func TestDetectionAnnotation_NotDetected(t *testing.T) {
	detected := types.DetectedProject{}
	for _, lang := range []string{"go", "javascript", "python", "rust", "java", "dotnet", "docker", "terraform"} {
		ann := devinit.ExportDetectionAnnotation(lang, detected)
		if ann != "" {
			t.Errorf("expected empty annotation for undetected %q, got %q", lang, ann)
		}
	}
}

func TestBuildLanguageOptions_DetectedFirst(t *testing.T) {
	detected := types.DetectedProject{
		HasDockerfile: true,
		HasGoMod:      true,
	}
	opts := devinit.ExportBuildLanguageOptions(detected)

	if len(opts) != 27 {
		t.Fatalf("expected 27 options, got %d", len(opts))
	}

	// First two should be the detected ones (Go and Docker, in canonical order)
	detectedOpts := []devinit.ExportLanguageOption{}
	nonDetectedOpts := []devinit.ExportLanguageOption{}
	for _, opt := range opts {
		if opt.Detected {
			detectedOpts = append(detectedOpts, opt)
		} else {
			nonDetectedOpts = append(nonDetectedOpts, opt)
		}
	}

	if len(detectedOpts) != 2 {
		t.Fatalf("expected 2 detected options, got %d", len(detectedOpts))
	}
	if len(nonDetectedOpts) != 25 {
		t.Fatalf("expected 25 non-detected options, got %d", len(nonDetectedOpts))
	}

	// Detected should come before non-detected in the full list
	lastDetectedIdx := -1
	firstNonDetectedIdx := len(opts)
	for i, opt := range opts {
		if opt.Detected {
			lastDetectedIdx = i
		} else if i < firstNonDetectedIdx {
			firstNonDetectedIdx = i
		}
	}
	if lastDetectedIdx >= firstNonDetectedIdx {
		t.Error("detected options should appear before non-detected options")
	}
}

func TestBuildLanguageOptions_Labels(t *testing.T) {
	detected := types.DetectedProject{HasGoMod: true}
	opts := devinit.ExportBuildLanguageOptions(detected)

	var goOpt devinit.ExportLanguageOption
	for _, opt := range opts {
		if opt.Value == "go" {
			goOpt = opt
			break
		}
	}

	if !strings.Contains(goOpt.Label, "Go") {
		t.Errorf("Go option label should contain 'Go', got %q", goOpt.Label)
	}
	if !strings.Contains(goOpt.Label, "(detected: go.mod)") {
		t.Errorf("Go option label should contain detection annotation, got %q", goOpt.Label)
	}
}

func TestBuildLanguageOptions_NoneDetected(t *testing.T) {
	detected := types.DetectedProject{}
	opts := devinit.ExportBuildLanguageOptions(detected)

	if len(opts) != 27 {
		t.Fatalf("expected 27 options, got %d", len(opts))
	}
	for _, opt := range opts {
		if opt.Detected {
			t.Errorf("no options should be detected, but %q is", opt.Value)
		}
	}
}

func TestPreSelectedLanguages(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:      true,
		HasCargoToml:  true,
		HasDockerfile: true,
	}
	selected := devinit.ExportPreSelectedLanguages(detected)

	if len(selected) != 3 {
		t.Fatalf("expected 3 pre-selected, got %d: %v", len(selected), selected)
	}
	expected := map[string]bool{"go": true, "rust": true, "docker": true}
	for _, s := range selected {
		if !expected[s] {
			t.Errorf("unexpected pre-selected language %q", s)
		}
	}
}

func TestPreSelectedLanguages_None(t *testing.T) {
	detected := types.DetectedProject{}
	selected := devinit.ExportPreSelectedLanguages(detected)

	if len(selected) != 0 {
		t.Errorf("expected 0 pre-selected for empty detection, got %d: %v", len(selected), selected)
	}
}

func TestQuickPathSummary_GoWithDirenvAndClaude(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
		Direnv:    true,
		ClaudeCode: true,
	}
	summary := devinit.ExportQuickPathSummary(answers)

	if !strings.Contains(summary, "Go 1.24") {
		t.Errorf("summary should contain 'Go 1.24', got %q", summary)
	}
	if !strings.Contains(summary, "devenv.sh") {
		t.Errorf("summary should contain 'devenv.sh', got %q", summary)
	}
	if !strings.Contains(summary, "direnv") {
		t.Errorf("summary should contain 'direnv', got %q", summary)
	}
	if !strings.Contains(summary, "Claude Code") {
		t.Errorf("summary should contain 'Claude Code', got %q", summary)
	}
}

func TestQuickPathSummary_MultiLanguage(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", Version: "22"},
		},
	}
	summary := devinit.ExportQuickPathSummary(answers)

	if !strings.Contains(summary, "Go 1.24") {
		t.Errorf("summary should contain 'Go 1.24', got %q", summary)
	}
	if !strings.Contains(summary, "JavaScript/TypeScript 22") {
		t.Errorf("summary should contain 'JavaScript/TypeScript 22', got %q", summary)
	}
	if !strings.Contains(summary, "devenv.sh") {
		t.Errorf("summary should contain 'devenv.sh', got %q", summary)
	}
}

func TestQuickPathSummary_NoLanguages(t *testing.T) {
	answers := types.WizardAnswers{}
	summary := devinit.ExportQuickPathSummary(answers)

	if summary != "devenv.sh" {
		t.Errorf("expected %q for no languages, got %q", "devenv.sh", summary)
	}
}

func TestQuickPathSummary_NoDirenvNoClaude(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "rust"},
		},
	}
	summary := devinit.ExportQuickPathSummary(answers)

	if strings.Contains(summary, "direnv") {
		t.Errorf("summary should not contain 'direnv', got %q", summary)
	}
	if strings.Contains(summary, "Claude Code") {
		t.Errorf("summary should not contain 'Claude Code', got %q", summary)
	}
	expected := "Rust + devenv.sh"
	if summary != expected {
		t.Errorf("expected %q, got %q", expected, summary)
	}
}
