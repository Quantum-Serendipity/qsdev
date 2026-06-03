package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestParseLocalConfig_FileNotFound(t *testing.T) {
	cfg, err := ParseLocalConfig("/nonexistent/path/.qsdev.local.yaml")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for missing file, got %v", cfg)
	}
}

func TestParseLocalConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev.local.yaml")
	content := `
extra_packages:
  - neovim
  - lazygit
claude_code:
  permission_level: permissive
tools:
  enabled:
    - changelog
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseLocalConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
		return
	}
	if len(cfg.ExtraPackages) != 2 {
		t.Errorf("expected 2 extra packages, got %d", len(cfg.ExtraPackages))
	}
	if cfg.ClaudeCode.PermissionLevel != "permissive" {
		t.Errorf("expected permissive, got %q", cfg.ClaudeCode.PermissionLevel)
	}
	if len(cfg.Tools.Enabled) != 1 || cfg.Tools.Enabled[0] != "changelog" {
		t.Errorf("expected [changelog], got %v", cfg.Tools.Enabled)
	}
}

func TestParseLocalConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev.local.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseLocalConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if cfg != nil {
		t.Error("expected nil config on error")
	}
}

func TestGenerateLocalTemplate_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	resolved := &types.QsdevConfig{}

	if err := GenerateLocalTemplate(dir, resolved); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".qsdev.local.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "Local developer overrides") {
		t.Error("expected template header in generated file")
	}
	if !strings.Contains(content, "extra_packages") {
		t.Error("expected extra_packages example in generated file")
	}
}

func TestGenerateLocalTemplate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev.local.yaml")

	// Write an existing file.
	existingContent := "# my custom config\n"
	if err := os.WriteFile(path, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// GenerateLocalTemplate should NOT overwrite.
	if err := GenerateLocalTemplate(dir, nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existingContent {
		t.Error("expected existing file to be preserved")
	}
}

func TestGenerateLocalTemplate_IncludesLanguageOverrides(t *testing.T) {
	dir := t.TempDir()
	resolved := &types.QsdevConfig{
		Languages: []types.LanguageConfig{
			{Name: "go", Version: "1.22"},
			{Name: "python", Version: "3.12"},
		},
	}

	if err := GenerateLocalTemplate(dir, resolved); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".qsdev.local.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "name: go") {
		t.Error("expected go language in template")
	}
	if !strings.Contains(content, "name: python") {
		t.Error("expected python language in template")
	}
}

func TestGenerateLocalTemplate_IncludesClaudeSection(t *testing.T) {
	dir := t.TempDir()
	enabled := true
	resolved := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled: &enabled,
		},
	}

	if err := GenerateLocalTemplate(dir, resolved); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".qsdev.local.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "claude_code:") {
		t.Error("expected claude_code section in template when Claude is enabled")
	}
}

func TestEnsureGitignoreEntry_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, ".qsdev.local.yaml") {
		t.Error("expected .qsdev.local.yaml in .gitignore")
	}
	if !strings.Contains(content, "# qsdev local configuration") {
		t.Error("expected section comment in .gitignore")
	}
}

func TestEnsureGitignoreEntry_AppendsWhenNotPresent(t *testing.T) {
	dir := t.TempDir()
	existing := "node_modules/\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "node_modules/") {
		t.Error("expected existing content to be preserved")
	}
	if !strings.Contains(content, ".qsdev.local.yaml") {
		t.Error("expected .qsdev.local.yaml to be appended")
	}
}

func TestEnsureGitignoreEntry_NoOpWhenPresent(t *testing.T) {
	dir := t.TempDir()
	existing := "node_modules/\n.qsdev.local.yaml\n.env\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureGitignoreEntry(dir, ".qsdev.local.yaml"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	// Should be unchanged.
	if string(data) != existing {
		t.Errorf("expected no changes, got %q", string(data))
	}
}
