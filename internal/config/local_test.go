package config

import (
	"os"
	"path/filepath"
	"testing"
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

// Regression: the local override file is decoded strictly, so a misspelled
// key (which would otherwise silently drop a local deny) is an error, while
// an empty or comments-only file is a valid empty override.
func TestParseLocalConfig_StrictKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"misspelled nested key", "tools:\n  disable: [semgrep]\n", true},
		{"misspelled top-level key", "tools:\n  disabled: [semgrep]\nsecurty:\n  level: baseline\n", true},
		{"valid keys", "tools:\n  disabled: [semgrep]\n", false},
		{"empty file", "", false},
		{"comments only", "# extra_packages:\n#   - neovim\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".qsdev.local.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := ParseLocalConfig(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got config %+v", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil config for an existing file")
			}
		})
	}
}
